# Getting Started Sample

Cluster with Tekton resources must be running. See [Deploy Cluster](../README.md#deploy-cluster).

Deploy or teardown pipelines, tasks and triggers.

```bash
# deploy
./gs.sh up

# teardown
./gs.sh down
```

## Test Triggers

We do it using curl and a simple http request.

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