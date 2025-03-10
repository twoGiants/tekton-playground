# Tekton Playground

Collecting Tekton pipelines and tasks for tests, experimentation and more.

## Prerequisites

- `kind`
- `docker`
- `kubectl`
- `tkn`
- `cosign`
- `jq`

## Deploy Cluster

Deploy or teardown kind with Tekton resources.

```bash
# deploy
./scripts/tk8.sh

# teardown but keep the registry for image caching
kind delete cluster -n tekton

# delete registry if you want
docker rm -f kind-registry
```

## Samples

1. [Getting Started Sample](docs/getting-started-sample.md)
1. [Chains Sample](docs/chains-sample.md)
