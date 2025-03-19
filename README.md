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

## Deploy Cluster

Deploy or teardown kind with Tekton resources.

```sh
# deploy
./scripts/tk8.sh

# teardown but keep the registry for image caching
kind delete cluster -n tekton

# delete registry if you want
docker rm -f kind-registry
```