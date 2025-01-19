# health-monitor-operator

The `health-monitor-operator` is an example kubernetes operator which I've implemented and tested in `minikube` with the help of [Github Copilot](https://github.com/copilot).

How it works
- The operator creates and manages deployment resources for a given `HealthCheck` CRD requests.
- It Watches on `HealthCheck.example.com` Kubernetes resource using the provided `service-account` credentials.
- On a `create` event, the operator generates a new `HealthCheck` deployment.
- On a `delete` event, it removes the corresponding `HealthCheck` deployment.
- Each HealthCheck CRD must include the following properties: `endpoint`, `intervalSeconds`, and `expectedStatus`.
- It uses a simple [HealthCheck-Scheduler](https://github.com/challamani/health-monitor) image in the generated deployment resource.

Use cases
- For organizations managing multiple microservices deployed across different clusters or on-premises environments, this operator can help monitor the health of each service by creating a dedicated health scheduler for them.
- By implementing a health monitor for individual services, you can establish an alerting system based on key metrics, enhancing observability and ensuring system reliability.


## Structure
```plaintext
health-monitor-operator/
├── cmd/
│   └── main.go         # Main entry point for the operator
│
├── controller/
│   └── controller.go   # Controller logic for HealthCheck CR.
│      
├── crd/
│   ├── crd.yaml        # Custom Resource Definition (CRD) for HealthCheck.
│   ├── rbac.yaml       # Role-based access control (RBAC) configuration.
│
├── deploy/
│   ├── deployment.yaml # Deployment resource of health-monitor-operator
│
├── examples/
│   ├── httpbin-healthcheck.yaml  # Simple CRD to generate traffic to httpbin endpoint.
│
├── deploy.sh          # To install operator & a simple health-check resource.
├── undeploy.sh        # To unstall operator & CRD. 
├── Dockerfile         # Dockerfile for building the operator image
├── go.mod             # Go module file
├── go.sum             # Go dependencies file
└── README.md          # Project README file
```

## Minikube
```shell
minikube start --kubernetes-version=v1.27.4 --driver=docker
```

## Build

```shell
go mod tidy
docker build -t health-monitor-operator:latest .
docker images
```

```shell
minikube image load health-monitor-operator:latest
```
Note: In deployment resource use `ImagePullPolicy: "Never"` if the image is already available in minikube node.


### Install 
```shell
#Install an example health-monitor-operator & simple healthcheck resource
./deploy.sh
```

## Uninstall
`./undeploy.sh`

## Example HealthCheck
```yaml
apiVersion: example.com/v1
kind: HealthCheck
metadata:
  name: example-healthcheck
spec:
  endpoint: "https://example.com/health"
  intervalSeconds: 60
  expectedStatus: 200
  auth: # TODO: not yet implemented this section.
    mtls:
      secretName: "mtls-secret"
    oauth:
      clientId: "example-client-id"
      clientSecret: "example-client-secret"
      tokenUrl: "https://example.com/token"
```

### Few additional commands

```shell
#remove the image from minikube node
minikube image rm health-monitor-operator:latest
```

## Execution Logs

```plaintext
mani@MacBook-Pro-2 health-monitor-operator % minikube image load health-monitor-operator:latest
╭──────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│                                                                                                          │
│    You are trying to run the amd64 binary on an M1 system.                                               │
│    Please consider running the darwin/arm64 binary instead.                                              │
│    Download at https://github.com/kubernetes/minikube/releases/download/v1.31.2/minikube-darwin-arm64    │
│                                                                                                          │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────╯
mani@MacBook-Pro-2 health-monitor-operator % 
mani@MacBook-Pro-2 health-monitor-operator % ./deploy.sh 
customresourcedefinition.apiextensions.k8s.io/healthchecks.example.com created
serviceaccount/health-monitor-operator created
secret/health-monitor-operator-secret created
role.rbac.authorization.k8s.io/health-monitor-operator created
rolebinding.rbac.authorization.k8s.io/health-monitor-operator created
clusterrole.rbac.authorization.k8s.io/health-monitor-operator-role created
clusterrolebinding.rbac.authorization.k8s.io/health-monitor-operator-binding created
deployment.apps/health-monitor-operator created
healthcheck.example.com/httpbin-healthcheck created
mani@MacBook-Pro-2 health-monitor-operator % 
mani@MacBook-Pro-2 health-monitor-operator % 
mani@MacBook-Pro-2 health-monitor-operator % k get pods -n test
NAME                                       READY   STATUS    RESTARTS   AGE
health-monitor-operator-6cf4f848d9-5tz5l   1/1     Running   0          29s
httpbin-healthcheck-84bfd775cb-9rjjl       1/1     Running   0          18s
mani@MacBook-Pro-2 health-monitor-operator % 
mani@MacBook-Pro-2 health-monitor-operator % k logs -f httpbin-healthcheck-84bfd775cb-9rjjl -n test

  .   ____          _            __ _ _
 /\\ / ___'_ __ _ _(_)_ __  __ _ \ \ \ \
( ( )\___ | '_ | '_| | '_ \/ _` | \ \ \ \
 \\/  ___)| |_)| | | | | || (_| |  ) ) ) )
  '  |____| .__|_| |_|_| |_\__, | / / / /
 =========|_|==============|___/=/_/_/_/
 :: Spring Boot ::                (v3.0.0)

2025-01-19T20:36:48.553Z  INFO 1 --- [           main] c.e.h.HealthMonitorApplication           : Starting HealthMonitorApplication v1.0.0 using Java 17.0.2 with PID 1 (/app/app.jar started by root in /app)
2025-01-19T20:36:48.554Z  INFO 1 --- [           main] c.e.h.HealthMonitorApplication           : No active profile set, falling back to 1 default profile: "default"
2025-01-19T20:36:49.448Z  INFO 1 --- [           main] o.s.b.w.embedded.tomcat.TomcatWebServer  : Tomcat initialized with port(s): 8080 (http)
2025-01-19T20:36:49.455Z  INFO 1 --- [           main] o.apache.catalina.core.StandardService   : Starting service [Tomcat]
2025-01-19T20:36:49.455Z  INFO 1 --- [           main] o.apache.catalina.core.StandardEngine    : Starting Servlet engine: [Apache Tomcat/10.1.1]
2025-01-19T20:36:49.529Z  INFO 1 --- [           main] o.a.c.c.C.[Tomcat].[localhost].[/]       : Initializing Spring embedded WebApplicationContext
2025-01-19T20:36:49.530Z  INFO 1 --- [           main] w.s.c.ServletWebServerApplicationContext : Root WebApplicationContext: initialization completed in 944 ms
2025-01-19T20:36:50.258Z  INFO 1 --- [           main] o.s.b.a.e.web.EndpointLinksResolver      : Exposing 14 endpoint(s) beneath base path '/actuator'
2025-01-19T20:36:50.352Z  INFO 1 --- [           main] o.s.b.w.embedded.tomcat.TomcatWebServer  : Tomcat started on port(s): 8080 (http) with context path ''
2025-01-19T20:36:50.391Z  INFO 1 --- [   scheduling-1] c.e.h.service.HealthCheckService         : Starting health check for endpoint: https://httpbin.org/status/201
2025-01-19T20:36:50.394Z  INFO 1 --- [           main] c.e.h.HealthMonitorApplication           : Started HealthMonitorApplication in 2.129 seconds (process running for 2.512)
2025-01-19T20:36:51.219Z  INFO 1 --- [   scheduling-1] c.e.h.service.HealthCheckService         : Health check successful for endpoint: https://httpbin.org/status/201. Status code: 201
2025-01-19T20:37:50.395Z  INFO 1 --- [   scheduling-1] c.e.h.service.HealthCheckService         : Starting health check for endpoint: https://httpbin.org/status/201
2025-01-19T20:37:50.945Z  INFO 1 --- [   scheduling-1] c.e.h.service.HealthCheckService         : Health check successful for endpoint: https://httpbin.org/status/201. Status code: 201
```