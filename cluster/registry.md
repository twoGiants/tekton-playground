# Registry 

**WIP:** In this setup multiple registries are configured and connected to the kind cluster. One for locally build images and one for each of the big public registries docker.io, quay.io, gcr.io and ghcr.io. Every registry is storing its data on the hosts filesystem in a mounted volume so you can safely stop and remove the registry containers when you delete the kind cluster without loosing the cached images and the need for a new download. This greatly reduces the startup time of the kind cluster, Tekton components and locally build images.

## Local

The registry stores its data in the `./cluster/data/local` directory of this project. The line `-v ./data/local:/var/lib/registry:z` below configures the registry to use a mounted volume on the host.The ending `:z` is needed on systems running with selinux enabled. Remove if needed.

```sh
docker run \
  -d \
  --restart=always \
  --name "${reg_name}" \
  -p "${reg_port}:5000" \
  -v ./data/local:/var/lib/registry:z \
  registry:2
```
Read the official docs [here](https://distribution.github.io/distribution/about/deploying/#storage-customization).

## Testing

Test connection to the registry from the cluster.

```sh
# pull from public registry
docker pull docker.io/library/busybox

# tag, the prefix tells podman to use local registry
docker tag busybox:latest localhost:5000/my-busybox

# push to local registry
docker push localhost:5000/my-busybox:latest

# deploy pod
kubectl apply -f cluster/testing/busybox-hello-pod.yaml

# watch pod
kubectl get po hello-busybox -w

# check log message: "Hello, Kubernetes!"
kubectl logs hello-busybox
```

## Podman

If you start a registry using podman you need to add an entry to `/etc/containers/registries.conf` or you wont be able to push to it.

```conf
[[registry]]
insecure = true 
location = "localhost:5000"
```

Test pushing to local registry and pulling from it.

```sh
# pull from public registry
podman pull docker.io/library/busybox

# tag, the prefix tells podman to use local registry
podman tag busybox:latest localhost:5000/my-busybox

# push to local registry
podman push localhost:5000/my-busybox:latest

# delete images
podman image remove busybox:latest
podman image remove localhost:5000/my-busybox

# pull from local registry
podman pull localhost:5000/my-busybox
```
