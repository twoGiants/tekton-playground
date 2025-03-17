# Tekton Developer Docs

## Kubernetes Objects

- they describe:
  - what containerized apps are running
  - their available resources
  - policies around them
- Kubernetes object = "record of intent" -> the Kubernetes system will work to ensure that the object exists; this is your cluster's desired state.
- e.g. Deployment, Pod, Service, Config Map

### Object Spec and Status
- spec: the desired state, provided by you
- status: current state, provided by the control plane

### Validation
```bash
kubectl --validate
```

### Important Links
- Official [docs](https://kubernetes.io/docs/concepts/overview/working-with-objects/).
- Required fields, see [Kubernetes API reference](https://kubernetes.io/docs/reference/kubernetes-api/).

## Tekton Components

### CRDs

Tekton objects like Tasks, TaskRuns, etc. are implemented as CRDs and defined [here](https://github.com/tektoncd/pipeline/tree/main/config) with the schemas in Go [here](https://github.com/tektoncd/pipeline/tree/main/pkg/apis/pipeline/v1).

### Controllers

> **Reconciling**: a [custom controller](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#custom-controllers) changes the clusters state based on an instance of a CRD.

Reconcilers change the cluster based on the desired behavior in an object's "spec", and update the object's "status" to reflect what happened.

### Webhooks

tba...

### Generated Code

tba...

### 