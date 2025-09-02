# Implantação de Aplicação Golang em Cluster Kubernetes

## Objetivo

Este guia demonstra como criar, containerizar e implantar uma aplicação web simples escrita em Golang em um cluster Kubernetes, utilizando Amazon ECR como repositório de imagens Docker.

## Pré-requisitos

- Docker Desktop instalado e configurado
- AWS CLI configurado com credenciais apropriadas
- Kubectl instalado e configurado para acessar seu cluster
- Acesso a um cluster Kubernetes (EKS ou outro)
- IDE (como Visual Studio Code)
- Privilégios administrativos na máquina local

## 1. Criação da Aplicação

### 1.1 Código da Aplicação (main.go)

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
)

func handler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Requisição recebida de %s para %s", r.RemoteAddr, r.URL.Path)
    hostname, err := os.Hostname()
    if err != nil {
        hostname = "desconhecido"
    }
    fmt.Fprintf(w, "Olá do Golang! Executando no pod: %s\n", hostname)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, "OK")
}

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    http.HandleFunc("/", handler)
    http.HandleFunc("/health", healthCheck)

    log.Printf("Servidor iniciando na porta %s", port)
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatalf("Erro ao iniciar o servidor: %v", err)
    }
}
```

Esta aplicação fornece:

- Um endpoint principal (`/`) que exibe uma mensagem com o hostname do pod
- Um endpoint de verificação de saúde (`/health`) para probes de Kubernetes
- Logs para facilitar a depuração

### 1.2 Dockerfile Multi-Estágio

```dockerfile
# Estágio de build - compila a aplicação
FROM golang:1.21-alpine AS builder

# Configura repositórios HTTP para evitar problemas de certificado SSL
RUN echo 'http://dl-cdn.alpinelinux.org/alpine/v3.18/main' > /etc/apk/repositories && \
    echo 'http://dl-cdn.alpinelinux.org/alpine/v3.18/community' >> /etc/apk/repositories

# Instala certificados e dependências necessárias
RUN apk update && apk add --no-cache ca-certificates

# Define o diretório de trabalho
WORKDIR /app

# Copia o código-fonte
COPY main.go .

# Compila para um binário estático
RUN CGO_ENABLED=0 GOOS=linux go build -o app main.go

# Estágio final - imagem mínima para execução
FROM alpine:3.18

# Configura repositórios HTTP
RUN echo 'http://dl-cdn.alpinelinux.org/alpine/v3.18/main' > /etc/apk/repositories && \
    echo 'http://dl-cdn.alpinelinux.org/alpine/v3.18/community' >> /etc/apk/repositories

# Instala certificados para suporte HTTPS
RUN apk update && apk add --no-cache ca-certificates

# Define o diretório de trabalho
WORKDIR /app

# Copia apenas o binário compilado do estágio anterior
COPY --from=builder /app/app .

# Expõe a porta da aplicação
EXPOSE 8080

# Define o comando de inicialização
CMD ["./app"]
```

Este Dockerfile utiliza uma abordagem multi-estágio para:

- Minimizar o tamanho da imagem final
- Separar o ambiente de compilação do ambiente de execução
- Evitar problemas com certificados SSL utilizando repositórios HTTP

## 2. Build e Publicação da Imagem

### 2.1 Construção da Imagem Docker

```bash
# Construir a imagem Docker on linux
docker build -t golang-exemplo:v1.0 .

# Construir a imagem Docker on Mac
docker build -t golang-exemplo:v1.0 --platform linux/amd64 .

# Verificar se a imagem foi criada corretamente
docker images | grep golang-exemplo
```

### 2.2 Publicação no Amazon ECR

```bash
# Criar tag para o repositório ECR
docker tag golang-exemplo:v1.0 060795900871.dkr.ecr.us-east-1.amazonaws.com/eks-service-images:golang-exemplo_v1

# Autenticar no ECR (executar como administrador)
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 060795900871.dkr.ecr.us-east-1.amazonaws.com

# Enviar imagem para o ECR
docker push 060795900871.dkr.ecr.us-east-1.amazonaws.com/eks-service-images:golang-exemplo_v1
```

> **Importante**: Execute o Docker Desktop e o terminal/VS Code como administrador para evitar problemas de autenticação com o ECR.

## 3. Implantação no Kubernetes

### 3.1 Manifesto Kubernetes (k8s-golang.yaml)

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: golang-exemplo
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: golang-app
  namespace: golang-exemplo
  labels:
    app: golang-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: golang-app
  template:
    metadata:
      labels:
        app: golang-app
    spec:
      containers:
        - name: golang-app
          image: 060795900871.dkr.ecr.us-east-1.amazonaws.com/eks-service-images:golang-exemplo_v1
          ports:
            - containerPort: 8080
          resources:
            limits:
              cpu: "200m"
              memory: "256Mi"
            requests:
              cpu: "100m"
              memory: "128Mi"
          env:
            - name: PORT
              value: "8080"
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: golang-app-service
  namespace: golang-exemplo
spec:
  selector:
    app: golang-app
  ports:
    - port: 80
      targetPort: 8080
  type: ClusterIP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: golang-app-ingress
  namespace: golang-exemplo
  annotations:
    kubernetes.io/ingress.class: "nginx"
spec:
  rules:
    - host: golang-app.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: golang-app-service
                port:
                  number: 80
```

Este manifesto cria:

- Um namespace dedicado (`golang-exemplo`)
- Um deployment com 3 réplicas e limites de recursos
- Probes de saúde para garantir alta disponibilidade
- Um serviço ClusterIP para comunicação interna
- Um Ingress para acesso externo (requer controlador NGINX)

### 3.2 Aplicando a Configuração

```bash
# Aplicar o manifesto Kubernetes
kubectl apply -f k8s-golang.yaml

# Verificar se o namespace foi criado
kubectl get namespaces | grep golang-exemplo

# Verificar se os pods estão em execução
kubectl get pods -n golang-exemplo

# Verificar status do deployment
kubectl get deployment -n golang-exemplo

# Verificar logs de um pod específico (substitua POD_NAME)
kubectl logs POD_NAME -n golang-exemplo
```

### 3.3 Teste de Acesso

```bash
# Encaminhamento de porta para teste local
kubectl port-forward svc/golang-app-service 8080:80 -n golang-exemplo

# Em outro terminal, teste o acesso
curl http://localhost:8080
curl http://localhost:8080/health
```

Alternativamente, você pode usar uma ferramenta como o Lens para gerenciar e testar sua aplicação:

1. Conecte-se ao cluster no Lens
2. Navegue até o namespace `golang-exemplo`
3. Selecione um pod e habilite o Port Forwarding
4. Acesse a aplicação através do navegador

## 4. Resolução de Problemas

### 4.1 Problemas Comuns

- **Erro de autenticação no ECR**

  - Solução: Execute Docker e terminal como administrador
  - Verifique credenciais AWS com `aws sts get-caller-identity`

- **Pods não iniciam (ImagePullBackOff)**

  - Solução: Verifique se a imagem existe no ECR
  - Verifique se o cluster tem permissão para acessar o ECR

- **Aplicação não responde**
  - Solução: Verifique logs dos pods
  - Verifique se os probes de saúde estão configurados corretamente

### 4.2 Comandos Úteis

```bash
# Verificar descrição detalhada de um pod
kubectl describe pod POD_NAME -n golang-exemplo

# Executar shell dentro de um pod
kubectl exec -it POD_NAME -n golang-exemplo -- /bin/sh

# Reiniciar deployment
kubectl rollout restart deployment golang-app -n golang-exemplo
```
