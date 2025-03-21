# Tekton Playground

Collecting Tekton knowledge, pipelines and tasks for tests, experimentation and more.

## Table of Contents

1. [Prerequisites](#prerequisites)
1. [Deploy Cluster](#deploy-cluster)
1. [Registry](cluster/registry.md)
1. [Tekton Developer Documentation](docs/tekton-dev-docs.md)
1. Samples
   - [Getting Started Sample](getting-started/getting-started-sample.md)
   - [Chains Sample](chains/chains-sample.md)

## Prerequisites

- `kind`
- `docker`
- `kubectl`
- `tkn`
- `cosign`
- `jq`
- `tailspin`

## Deploy Cluster

Deploy or teardown kind with Tekton resources.

```sh
# deploy kind with Tekton
./scripts/tk8.sh

# deploy only kind
./scripts/tk8.sh -s

# teardown but keep the registry for image caching
kind delete cluster -n tekton

# delete registry if you want
docker rm -f kind-registry
```

## Deploy Local Tekton

Deploy a Kind cluster with a local registry. Then go to your `tekton-pipeline` fork clone directory.

```sh
# setup ko to use the local registry
export KO_DOCKER_REPO="localhost:5000"

# install pipeline
ko apply -R -f config/

# verify installation
kubectl get pods -n tekton-pipelines

# delete but keep the namespace
ko delete -f config/

# delete all Tekton components => will also delete Dashboard
ko delete -R -f config/
```

## Develop

If you make changes to the code, redeploy the controller.

```sh
ko apply -f config/controller.yaml
```

Access the logs from the controller or webhook colorized by tailspin.

```sh
kubectl -n tekton-pipelines logs $(kubectl -n tekton-pipelines get pods -l app=tekton-pipelines-controller -o name) | tspin

kubectl -n tekton-pipelines logs $(kubectl -n tekton-pipelines get pods -l app=tekton-pipelines-webhook -o name) | tspin
```
