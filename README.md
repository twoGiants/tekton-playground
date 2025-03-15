# Tekton Playground

Collecting Tekton knowledge, pipelines and tasks for tests, experimentation and more.

## Table of Contents

1. Samples
   - [Getting Started Sample](getting-started/getting-started-sample.md)
   - [Chains Sample](chains/chains-sample.md)
1. [Prerequisites](#prerequisites)
1. [Deploy Cluster](#deploy-cluster)
1. [Registry](cluster/registry.md)
1. Tekton Developer Documentation
   - [Kubernetes Objects](#kubernetes-objects)

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
## Kubernetes Objects

**Kubernetes object**
- they describe:
  - what containerized apps are running
  - their available resources
  - policies around them
- Kubernetes object = "record of intent" -> the Kubernetes system will work to ensure that the object exists; this is your cluster's desired state.
- e.g. Deployment, Pod, Service, Config Map

**Object spec and status**
- spec: the desired state, provided by you
- status: current state, provided by the control plane

**Validation**
```bash
kubectl --validate
```

**Important Links**
- Official [docs](https://kubernetes.io/docs/concepts/overview/working-with-objects/).
- Required fields, see [Kubernetes API reference](https://kubernetes.io/docs/reference/kubernetes-api/).

## Tekton Components

**CRDs**

tba...

**Controllers**

tba...

**Webhooks**

tba...

**Generated Code**

tba...