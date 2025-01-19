kubectl apply -f crd/crd.yaml
kubectl apply -f crd/rbac.yaml
kubectl apply -f deploy/deployment.yaml

sleep 10
kubectl apply -f examples/healthcheck.yaml