# Tekton Playground

Collecting Tekton knowledge, pipelines and tasks for tests, experimentation and more.

## Table of Contents

1. [Prerequisites](#prerequisites)
1. [Deploy Cluster](#deploy-cluster)
1. [Registry](cluster/registry.md)
1. [Tekton Developer Documentation](docs/tekton-dev-docs.md)
1. Samples
   - [Getting Started Sample](samples/getting-started/getting-started-sample.md)
   - [Chains Sample](samples/chains/chains-sample.md)
1. [Reliable Distributed Systems](docs/reliable-distributed-systems.md)

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
./cluster/cluster.sh

# deploy only kind
./cluster/cluster.sh -s

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

Execute a hello-world `PipelineRun` and follow the logs:

```sh
kubectl create -f - <<EOF
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: hello-world-run
spec:
  pipelineSpec:
    tasks:
    - name: hello
      taskSpec:
        steps:
        - name: echo
          image: alpine
          script: |
            #!/bin/sh
            echo "Hello World"
EOF

kubectl logs -f hello-world-run-hello-pod
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
