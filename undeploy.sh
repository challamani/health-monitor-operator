#!/bin/sh

kubectl delete -f examples/httpbin-healthcheck.yaml
kubectl delete -f deploy/deployment.yaml
kubectl delete -f crd/rbac.yaml
kubectl delete -f crd/crd.yaml