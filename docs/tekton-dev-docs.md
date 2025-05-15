# Tekton Developer Docs

## Kubernetes Objects

- they describe:
  - what containerized apps are running
  - their available resources
  - policies around them
- Kubernetes object = "record of intent" -> the Kubernetes system will work to ensure that the object exists; this is your cluster's desired state.
- e.g. Deployment, Pod, Service, Config Map

### Object Spec and Status

- `spec`: the desired state, provided by you
- `status`: current state, provided by the control plane

### Validation

```sh
kubectl --validate
```

### Important Links

- Official [docs](https://kubernetes.io/docs/concepts/overview/working-with-objects/).
- Required fields, see [Kubernetes API reference](https://kubernetes.io/docs/reference/kubernetes-api/).

## Tekton Components

### CRDs

Tekton objects like `Tasks`, `TaskRuns`, etc. are implemented as CRDs and defined [here](https://github.com/tektoncd/pipeline/tree/main/config) with the schemas in Go [here](https://github.com/tektoncd/pipeline/tree/main/pkg/apis/pipeline/v1).

[CRD Tutorial](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/).

### Controllers

In a simplified way, Kubernetes works by allowing us to declare the desired state of our system, and then its controllers continuously observe the cluster and take actions to ensure that the actual state matches the desired state.

> **Reconciling**: a [custom controller](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#custom-controllers) changes the clusters state based on an instance of a CRD.

*Reconcilers* change the cluster based on the desired behavior in an object's `spec`, and update the object's `status` to reflect what happened.

Not all Tekton CRDs use controllers. There is no *reconciler* for `Tasks`, you need to use a `TaskRun` which is executed by a `TaskRun` *reconciler*.

`TaskRun` *reconciler* is [here](https://github.com/tektoncd/pipeline/blob/main/pkg/reconciler/taskrun/taskrun.go) and PipelineRun [here](https://github.com/tektoncd/pipeline/blob/main/pkg/reconciler/pipelinerun/pipelinerun.go).

Build a controller using Kubebuilder [tutorial](https://book.kubebuilder.io/introduction.html) (Tekton uses Knative).

### Admission Webhooks

Tekton CRDs use validating and some mutating admission webhooks.

[Admission webhooks](https://web.archive.org/web/20230928184501/https://banzaicloud.com/blog/k8s-admission-webhooks/) in-depth.

### Generated Code

[This](https://github.com/tektoncd/pipeline/blob/main/docs/developers/controller-logic.md#generated-code) needs more clarification once development is started.

## Technical Deep Dive

### Tasks

- each step is a kubernetes container
- `script` field is not available => tekton extend kubernetes containers
- `TaskRun` creates a pod and runs each step as a container in that pod
- get the pod name which the `TaskRun` created

```sh
kubectl get -o yaml taskrun "<task-run-name>" | less
```

- you can embed `Tasks` in `TaskRuns`
- k8 starts containers in a pod at once but tekton wants the step containers executed one after another -> realized through `entrypoint` logic

### TaskRun Controller

A controller is an object which encapsulates all the required resources during the reconcile loop. [Here](https://github.com/tektoncd/pipeline/blob/b7a37285e85090ecbcd70ebeba97eb5ddfeb8ad5/pkg/reconciler/taskrun/controller.go#L55) is the `TaskRun` controller:

```go
func NewController(
  opts *pipeline.Options, 
  clock clock.PassiveClock,
) func(context.Context, configmap.Watcher) *controller.Impl {
  return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
    logger := logging.FromContext(ctx)
    kubeclientset := kubeclient.Get(ctx)
    pipelineclientset := pipelineclient.Get(ctx)
    taskRunInformer := taskruninformer.Get(ctx)
    podInformer := filteredpodinformer.Get(ctx, v1.ManagedByLabelKey)

    // ...

    if _, err := podInformer.Informer().AddEventHandler(
      cache.FilteringResourceEventHandler{
        FilterFunc: controller.FilterController(&v1.TaskRun{}),
        Handler:    controller.HandleAll(impl.EnqueueControllerOf),
      },
    ); err != nil {
      logging.FromContext(ctx).Panicf("Couldn't register Pod informer event handler: %w", err)
    }

    // ...
```

We see a collection of objects called *informers*. An *informer* is used by a controller to listen for changes in the status of resources. We are adding event listeners which listen for changes in the Pod and they filter those changes to those which are related to the `TaskRun` controller. In the diagram below this is the last step *"K8s notifies TaskRun reconciler"*. Here we're registering to receive those events.

[Here](https://github.com/tektoncd/pipeline/blob/b7a37285e85090ecbcd70ebeba97eb5ddfeb8ad5/pkg/reconciler/taskrun/taskrun.go#L115) is the `TaskRun` *reconciler*:

```go
func (c *Reconciler) ReconcileKind(ctx context.Context, tr *v1.TaskRun) pkgreconciler.Event {
  // ...
  
  // Reconcile this copy of the task run and then write back any status
  // updates regardless of whether the reconciliation errored out.
  if err = c.reconcile(ctx, tr, rtr); err != nil {
    logger.Errorf("Reconcile: %v", err.Error())
    // ...
  }
}

// ...

func (c *Reconciler) reconcile(ctx context.Context, tr *v1.TaskRun, rtr *resources.ResolvedTask) error {
  // ...
}
```

The `ReconcileKind` method is the first one which is called when the Pod and `TaskRun` updates are observed. The `reconcile` method does most of the leg work when it comes to creating Pods or looking at the differences in the Pod and the `TaskRun`.

How does a controller turn a `TaskRun` into a Pod? Inside the controller is a *reconciler* which implements a reconcile loop. It's job is to turn the YAML description and turn it into a k8 Pod. This is how it looks.

```plaintext
 |-> TaskRun reconciler gets notified of new TaskRun
 |              ↓
 |   Receives TaskRun data and looks up associated Task
 |              ↓
 |   Converts Task and TaskRun to Pod yaml using their fields
 |              ↓
 |   Submits Pod to k8s
 |              ↓
 |   Pod executes containers, changes status, emits events
 |              ↓
 |-- Changes are reported to k8s. K8s notifies TaskRun reconciler
```

Next is the loop when the reconciler gets notified when a pod already exists and it gets notified about its status.

```plaintext
 |-> TaskRun reconciler gets notified of Pod updates
 |              ↓
 |   Looks up Pod data
 |              ↓
 |   "Reconciles" Pod state with TaskRun state
 |              ↓
 |   Records new status in TaskRun
 |              ↓
 |   Pod continues executing, emitting events
 |              ↓
 |-- Events and status updates reported to k8s. K8s notifies TaskRun reconciler
```

In the step *"Reconciles Pod state with TaskRun state"* the *reconciler* looks at the difference between the Pod state and the `TaskRun` state and  then updates the `TaskRun` state to reflect the Pod state.

### Example

An example implementation of a CRD with a controller and a *reconciler* using the [kubebuilder](https://book.kubebuilder.io/getting-started) framework can be found in this repository in[samples/memcached-operator](../samples/memcached-operator/README.md).
