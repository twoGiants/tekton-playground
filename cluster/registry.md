# Registry

In this setup multiple registries are configured and connected to the kind cluster. One for locally build images and one for each of the big public registries docker.io, quay.io, gcr.io and ghcr.io. Every registry is storing its data on the hosts filesystem in a mounted volume so you can safely stop and remove the registry containers when you delete the kind cluster without loosing the cached images and the need for a new download. This greatly reduces the startup time of the kind cluster, Tekton components and locally build images.

## Local

The registry stores its data in the `./cluster/data/local` directory of this project. The line `-v ./data/local:/var/lib/registry:z` below configures the registry to use a mounted volume on the host.The ending `:z` is needed on systems running with selinux enabled. Remove if needed.

```bash
docker run \
  -d \
  --restart=always \
  --name "${reg_name}" \
  -p "${reg_port}:5000" \
  -v ./data/local:/var/lib/registry:z \
  registry:2
```
Read the official docs [here](https://distribution.github.io/distribution/about/deploying/#storage-customization).