package mocks

var MockDatadogHelm = `
datadog:
  apiKeyExistingSecret: "datadog-secret"
  site: "datadoghq.com"
  clusterName: local-dev-tom
  hostname: "kind-tom"

  # Adicione estas configurações
  env:
    - name: DD_CLOUD_PROVIDER_METADATA
      value: "[]"
    - name: DD_CONTAINER_CGROUP_ROOT
      value: "/host/sys/fs/cgroup/"
    - name: DD_KUBELET_TLS_VERIFY
      value: "false"

  # Desabilitar a detecção automática de hostname
  dogstatsd:
    useHostPort: true

  # Para o problema de cgroup no Docker Desktop
  kubelet:
    tlsVerify: false
  `

var MockDatadogOperator = `
apiVersion: datadoghq.com/v2alpha1
kind: DatadogAgent
metadata:
  name: datadog
spec:
  global:
    clusterName: local-dev-tom 
    site: "datadoghq.com"
    credentials:
      apiSecret:
        secretName: datadog-secret
        keyName: api-key
    kubelet:
      tlsVerify: false
  `

var MockNginx = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: webhook-agent
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx-container
        image: nginx:latest
        ports:
        - containerPort: 80
  `
