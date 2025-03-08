# Tekton Playground

Collecting Tekton pipelines and tasks for tests, experimentation and more.

## Getting Started

```bash
# deploy
./scripts/tp.sh up

# teardown
./scripts/tp.sh down

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