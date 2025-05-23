# Memcached Operator

## Purpose

This a sample implementation of a Kubernetes controller with a reconciler using the Kubebuilder framework. Additional tests were implemented to have a 100% (WIP: 79.2%) test coverage.

### Reconciliation Process

Pseudo-code example of the loop:

```go
reconcile App {
  // Check if a Deployment for the app exists, if not, create one
  // If there's an error, then restart from the beginning of the reconcile
  if err != nil {
    return reconcile.Result{}, err
  }

  // Check if a Service for the app exists, if not, create one
  // If there's an error, then restart from the beginning of the reconcile
  if err != nil {
    return reconcile.Result{}, err
  }

  // Look for Database CR/CRD
  // Check the Database Deployment's replicas size
  // If deployment.replicas size doesn't match cr.size, then update it
  // Then, restart from the beginning of the reconcile. For example, by returning `reconcile.Result{Requeue: true}, nil`.
  if err != nil {
    return reconcile.Result{Requeue: true}, nil
  }

  // If at the end of the loop:
  // Everything was executed successfully, and the reconcile can stop
  return reconcile.Result{}, nil
}
```

Return options:

- with error:

    ```go
    return ctrl.Result{}, err
    ```

- without error:

    ```go
    return ctrl.Result{Requeue: true}, nil
    ```

- stop the reconcile loop:

    ```go
    return ctrl.Result{}, nil
    ```

- reconcile again after `X` time:

    ```go
    return ctrl.Result{RequeueAfter: nextRun.Sub(r.Now())}, nil
    ```

## Getting Started

### Prerequisites

- go version v1.23.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

> **NOTE:** Use the `KinD` [cluster](/README.md#deploy-cluster) with a local [registry](/cluster/registry.md) from this repository to deploy the operator.

### To Deploy on the cluster

**Build and push your image to the location specified by `IMG`:**

```sh
# your setup
make docker-build docker-push IMG=<some-registry>/memcached-operator:tag

# KinD cluster setup from this repo
make docker-build docker-push IMG=localhost:5000/controller:latest
```

> **NOTE:** This image ought to be published in the personal registry you specified.
> And it is required to have access to pull the image from the working environment.
> Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
# your setup
make deploy IMG=<some-registry>/memcached-operator:tag

# KinD cluster setup from this repo
make deploy IMG=localhost:5000/controller:latest
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin privileges or be logged in as admin.

**Check instance of operator is running in your cluster:**

```sh
kubectl get all -n memcached-operator-system

NAME                                                         READY   STATUS    RESTARTS   AGE
pod/memcached-operator-controller-manager-5c7d9c7f64-782bx   1/1     Running   0          24s

NAME                                                            TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)    AGE
service/memcached-operator-controller-manager-metrics-service   ClusterIP   10.96.21.156   <none>        8443/TCP   24s

NAME                                                    READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/memcached-operator-controller-manager   1/1     1            1           24s

NAME                                                               DESIRED   CURRENT   READY   AGE
replicaset.apps/memcached-operator-controller-manager-5c7d9c7f64   1         1         1       24s
```

**Create instances of your solution:**

You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

**Follow the logs:**

```sh
kubectl logs -f -n memcached-operator-system memcached-operator-controller-manager-5c7d9c7f64-782bx
```

**Test the reconciler is working by manually changing the replicas:**

```sh
kubectl edit deploy memcached-sample
```

**Observe the log message of changing back the size:**

```sh
2025-05-20T09:11:08Z    INFO    found diverging size (4), changing back to (1)  {"controller": "memcached", "controllerGroup": "cache.example.com", "controllerKind": "Memcached", "Memcached": {"name":"memcached-sample","namespace":"default"}, "namespace": "default", "name": "memcached-sample", "reconcileID": "1c2d6d0b-70f0-4cc5-9a8a-8b7426e30d3c", "Deployment.Namespace": "default", "Deployment.Name": "memcached-sample"}

```

### To Uninstall

**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Test

**Prepare you environment to run the tests:**

```sh
make setup-envtest
```

**Run all the tests:**

```sh
make test
```

**Check coverage in browser:**

```sh
go tool cover -html=cover.out
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

    ```sh
    make build-installer IMG=<some-registry>/memcached-operator:tag
    ```

    **NOTE:** The makefile target mentioned above generates an 'install.yaml'
    file in the dist directory. This file contains all the resources built
    with Kustomize, which are necessary to install this project without its
    dependencies.

2. Using the installer

    Users can just run `kubectl apply -f <URL for YAML BUNDLE>` to install
    the project, i.e.:

    ```sh
    kubectl apply -f https://raw.githubusercontent.com/<org>/memcached-operator/<tag or branch>/dist/install.yaml
    ```

### By providing a Helm Chart

1. Build the chart using the optional helm plugin

    ```sh
    kubebuilder edit --plugins=helm/v1-alpha
    ```

2. See that a chart was generated under 'dist/chart', and users
can obtain this solution from there.

    **NOTE:** If you change the project, you need to update the Helm Chart
    using the same command above to sync the latest changes. Furthermore,
    if you create webhooks, you need to use the above command with
    the '--force' flag and manually ensure that any custom configuration
    previously added to 'dist/chart/values.yaml' or 'dist/chart/manager/manager.yaml'
    is manually re-applied afterwards.

> **NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

<http://www.apache.org/licenses/LICENSE-2.0>

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
