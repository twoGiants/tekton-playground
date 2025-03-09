# Tekton Playground

Collecting Tekton pipelines and tasks for tests, experimentation and more.

## Getting Started

Deploy or teardown kind with Tekton resources.

```bash
# deploy
./scripts/tk8.sh

# teardown
kind delete cluster -n tekton && docker rm -f kind-registry
```

Deploy or teardown pipelines, tasks and triggers.

```bash
# deploy
./scripts/tp.sh up

# teardown
./scripts/tp.sh down
```

Test triggers.

```bash
# enable port forwarding to be able to hit the event listener
kubectl port-forward service/el-hello-listener 8080

# trigger pipeline run via http request
curl -v \
   -H 'content-Type: application/json' \
   -d '{"username": "Tekton"}' \
   http://localhost:8080

# check logs
tkn pr logs hello-goodbye-run-
```