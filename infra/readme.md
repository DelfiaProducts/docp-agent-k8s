# Guide

- **Test Windows**

```powershell
kubectl apply -k .\namespaces
#kubectl create configmap webhook-code --from-file=.\resources\app.js -n webhook-agent
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.17.0/cert-manager.yaml
kubectl apply -k .\resources
# After done
kubectl apply -k .\sample_pods
```

- **Clean up Windows**

```powershell
kubectl delete -k .\sample_pods
kubectl delete -k .\resources
#kubectl delete configmap webhook-code -n webhook-agent
kubectl delete -k .\namespaces
```

- **Test Linux/Mac**

```shell
kubectl apply -k ./namespaces
#kubectl create configmap webhook-code --from-file=./resources/app.js -n webhook-agent
kubectl apply -k ./resources
# After done
kubectl apply -k ./sample_pods
```

- **Clean up Linux/Mac**

```shell
kubectl delete -k ./sample_pods
kubectl delete -k ./resources
#kubectl delete configmap webhook-code -n webhook-agent
kubectl delete -k ./namespaces
```

### Create Cluster Local

Usando Kind(kubernetes in docker)

Criando Cluster

```
kind create cluster --name local --config infra/cluster_local/kind-cluster.yaml
```

Create Certificate

```
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.17.0/cert-manager.yaml
```

Move images for nodes with `kind`

```
kind load docker-image k8s-agent:v1.0 --name local

```
