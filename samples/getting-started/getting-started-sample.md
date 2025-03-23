# Getting Started Sample

Cluster with Tekton resources must be running. See [Deploy Cluster](../README.md#deploy-cluster).

Deploy or teardown pipelines, tasks and triggers.

```sh
# deploy
./gs.sh up

# teardown
./gs.sh down
```

## Test Tasks

Run the hello world task.

```sh
kubectl create -f getting-started/runs/hello-world-run.yaml
```

Check its logs.

```sh
tkn tr logs -f hello-task-run-...
```

Get infos about the last task run.

```sh
tkn tr describe --last
```

## Test Triggers

We do it using curl and a simple http request.

```sh
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