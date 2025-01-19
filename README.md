# health-monitor-operator
health-monitor-operator


## Build
```shell
go mod tidy
```

## Minikube
```shell
minikube start --kubernetes-version=v1.27.4 --driver=docker
```

```shell
docker build -t health-monitor-operator:latest .
docker images
```

```shell
minikube image load health-monitor-operator:latest
```

```yaml
apiVersion: example.com/v1
kind: HealthCheck
metadata:
  name: example-healthcheck
spec:
  endpoint: "https://example.com/health"
  intervalSeconds: 60
  expectedStatus: 200
  auth:
    mtls:
      secretName: "mtls-secret"
    oauth:
      clientId: "example-client-id"
      clientSecret: "example-client-secret"
      tokenUrl: "https://example.com/token"
```