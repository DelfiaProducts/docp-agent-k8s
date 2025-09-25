# agent-k8s

Este projeto é um agente para gerenciamento de provedores de monitoramento para kubernetes.

## Funcionalidades

- Monitoramento de agentes
- Execução de comandos remotos
- Integração com outros serviços

## Requisitos

- Go 1.24+
- Dependências listadas em `go.mod`
- Utility `make` installed

## Instalação

```bash
git clone https://github.com/OryaHub/agent-k8s.git
cd agent-k8s
```
## Construção

```bash
REPOSITORY=[repository-name] TAG=[tag] PLATFORM=[platform] DOCKERFILE=[dockerfile] make build 
```
