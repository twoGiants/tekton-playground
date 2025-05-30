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

A controller is an object which encapsulates all the required resources during the reconcile loop. [`TaskRun` controller code](https://github.com/tektoncd/pipeline/blob/b7a37285e85090ecbcd70ebeba97eb5ddfeb8ad5/pkg/reconciler/taskrun/controller.go#L55) is here:

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

The [`TaskRun` *reconciler* code](https://github.com/tektoncd/pipeline/blob/b7a37285e85090ecbcd70ebeba97eb5ddfeb8ad5/pkg/reconciler/taskrun/taskrun.go#L115) is here:

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

#### High Level Overview

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

#### Low Level Overview

Lets dive deeper into the implementation. The comments in the code give already a good explanation what happens step by step.

**Check if `TaskRun` has started in [line 131](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L131):**

```go
 if !tr.HasStarted() {
  tr.Status.InitializeConditions()

  if tr.Status.StartTime.Sub(tr.CreationTimestamp.Time) < 0 {
   logger.Warnf(...)
   tr.Status.StartTime = &tr.CreationTimestamp
  }

  afterCondition := tr.Status.GetCondition(apis.ConditionSucceeded)
  events.Emit(ctx, nil, afterCondition, tr)
 }
```

And if it didn't initialize the status, fix timestamp issues and an user facing event.

**Check if `TaskRun` is complete in [line 148](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L148):**

```go
 if tr.IsDone() {
  logger.Infof("taskrun done : %s \n", tr.Name)

  tr.SetDefaults(ctx)

  useTektonSidecar := true
  if config.FromContextOrDefaults(ctx).FeatureFlags.EnableKubernetesSidecar {
  // ...
  }

  return c.finishReconcileUpdateEmitEvents(ctx, tr, before, nil)
 }
```

And if it is ensure default values are set and the sidecar stoppage is handled by Tekton or K8 if supported. At the end perform a set of operations in a method like status updates and emitting events.

Then we check for canceled ond timed out TaskRun and if we have pod failures starting in [line 177](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L177):

```go
 if tr.IsCancelled() {
  // ...
 }

 if tr.HasTimedOut(ctx, c.Clock) {
    // ...
 }

 if failed, reason, message := c.checkPodFailed(ctx, tr); failed {
    // ...
 }

```

**Now prepare for the actual run in the `prepare` method in [line 401](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L401):**

```go
 vp, err := c.verificationPolicyLister.VerificationPolicies(tr.Namespace).List(labels.Everything())
 if err != nil {
  return nil, nil, fmt.Errorf("...")
 }
```

Pulls in policies for Tekton Chains.

```go
 getTaskfunc := resources.GetTaskFuncFromTaskRun(...)
 taskMeta, taskSpec, err := resources.GetTaskData(ctx, tr, getTaskfunc)

```

Resolves the actual `Task` which could be from a remote source like GitHub.

```go
 switch {
  //  ...
 default:
  if err := storeTaskSpecAndMergeMeta(ctx, tr, taskSpec, taskMeta); err != nil {
   logger.Errorf(...)
  }
 }
```

Handles errors and stores the fetched `TaskSpec` for auditability.

**Perform the same steps for `StepActions` in [line 439](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L439):**

```go
 steps, err := resources.GetStepActionsData(...)
 switch {
  // ...
 default:
  taskSpec.Steps = steps
  if err := storeTaskSpecAndMergeMeta(ctx, tr, taskSpec, taskMeta); err != nil {
   logger.Errorf(...)
  }
 }
```

Fetch all `StepActions`, handle errors and update `TaskSpec` with resolved steps.

**Perform signature checks if Tekton Chains was used in [line 465](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L465):**

```go
if taskMeta.VerificationResult != nil {
  switch taskMeta.VerificationResult.VerificationResultType {
  case trustedresources.VerificationError:
   logger.Errorf(...)
   tr.Status.MarkResourceFailed(...)
   tr.Status.SetCondition(&apis.Condition{
    Type:    trustedresources.ConditionTrustedResourcesVerified,
    Status:  corev1.ConditionFalse,
    Message: taskMeta.VerificationResult.Err.Error(),
   })
   return ..., controller.NewPermanentError(taskMeta.VerificationResult.Err)
  }
 }
```

**Build the `ResolvedTask` for downstream use and perform validations in [line 492](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L492):**

```go
rtr := &resources.ResolvedTask{
  TaskName: taskMeta.Name,
  TaskSpec: taskSpec,
  Kind:     resources.GetTaskKind(tr),
 }

 if err := validateTaskSpecRequestResources(taskSpec); err != nil {
  //  ..
  return nil, nil, controller.NewPermanentError(err)
 }

 if err := ValidateResolvedTask(ctx, tr.Spec.Params, &v1.Matrix{}, rtr); err != nil {
  //  ..
 }

 if config.FromContextOrDefaults(ctx).FeatureFlags.EnableParamEnum {
  if err := ValidateEnumParam(ctx, tr.Spec.Params, rtr.TaskSpec.Params); err != nil {
    //  ..
  }
 }

 if err := resources.ValidateParamArrayIndex(rtr.TaskSpec, tr.Spec.Params); err != nil {
    //  ..
 }
```

Validate requested resources, the resolved task, enum parameters and parameter array index.

**Prepare workspaces in [line 518](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L524):**

```go
 if err := c.updateTaskRunWithDefaultWorkspaces(ctx, tr, taskSpec); err != nil {
  // ...
 }

 var workspaceDeclarations []v1.WorkspaceDeclaration
 if tr.Spec.TaskSpec != nil {
  for _, ws := range tr.Spec.Workspaces {
   wspaceDeclaration := v1.WorkspaceDeclaration{Name: ws.Name}
   workspaceDeclarations = append(workspaceDeclarations, wspaceDeclaration)
  }
  workspaceDeclarations = append(workspaceDeclarations, taskSpec.Workspaces...)
 } else {
  workspaceDeclarations = taskSpec.Workspaces
 }
 if err := workspace.ValidateBindings(ctx, workspaceDeclarations, tr.Spec.Workspaces); err != nil {
  // ...
 }
```

Add workspaces from config defaults and validate user workspace bindings for correct volume mounting later.

**Validate affinity assistant in [line 549](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L549):**

```go
 aaBehavior, err := affinityassistant.GetAffinityAssistantBehavior(ctx)
 if err != nil {
  return nil, nil, controller.NewPermanentError(err)
 }
 if aaBehavior == affinityassistant.AffinityAssistantPerWorkspace {
  if err := workspace.ValidateOnlyOnePVCIsUsed(tr.Spec.Workspaces); err != nil {
    // ...
  }
 }
```

Remember, [Tekton's Affinity Assistant](https://tekton.dev/docs/pipelines/affinityassistants/) schedules `PipelineRun` `Pods` to the same node so that `TaskRuns` execute parallel while sharing volume.

Make sure if `AffinityAssistantPerWorkspace` is set that each workspace uses only one PVC per `TaskRun`. Because multi-PVC workspaces break the guarantee that all pods can access the same volume on the same node.

**Handle step and sidecar overrides [line 561](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L561):**

```go
 if err := validateOverrides(taskSpec, &tr.Spec); err != nil {
  // ...
 }
```

This is a beta feature (which is not even documented) where you can override `Step` and `Sidecar` properties of a `Task` in a `TaskRun`. Here it is checked that the override refers to an existing `Step/Sidecar`.

**Next important method is `reconcile`  in [line 214](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L214):**

```go
 if err = c.reconcile(ctx, tr, rtr); err != nil {
  logger.Errorf("Reconcile: %v", err.Error())
  if errors.Is(err, sidecarlogresults.ErrSizeExceeded) {
   cfg := config.FromContextOrDefaults(ctx)
   message := fmt.Sprintf(....)
   err := c.failTaskRun(...)
   return c.finishReconcileUpdateEmitEvents(ctx, tr, before, err)
  }
 }
```

It is responsible for creating or finding the `Pod` that runs this `TaskRun` and update the `TaskRun`'s status to reflect what happened in the cluster.

**If `Pod` is already tracked, get it in [line 587](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L587):** 

```go
 var pod *corev1.Pod

 if tr.Status.PodName != "" {
  pod, err = c.podLister.Pods(tr.Namespace).Get(tr.Status.PodName)
  if k8serrors.IsNotFound(err) {
   // Keep going, this will result in the Pod being created below.
  } else if err != nil {
   logger.Errorf("Error getting pod %q: %v", tr.Status.PodName, err)
   return err
  }
 } else {
  labelSelector := labels.Set{pipeline.TaskRunLabelKey: tr.Name}
  pos, err := c.podLister.Pods(tr.Namespace).List(labelSelector.AsSelector())
  // ...
 }
```

If the pod is missing move on, it will be created later. Otherwise find `Pod` by label with the `TaskRun`'s name.

**Then handle workspace PVCs in [line 617](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L617):**

```go
 if pod == nil && tr.HasVolumeClaimTemplate() {
  for _, ws := range tr.Spec.Workspaces {
   if err := c.pvcHandler.CreatePVCFromVolumeClaimTemplate(...); err != nil {
    // ...
    return controller.NewPermanentError(err)
   }
  }

  taskRunWorkspaces := applyVolumeClaimTemplates(...)
  tr.Spec.Workspaces = taskRunWorkspaces
 }
```

If the pod doesn't exist and the `TaskRun` has a workspace template, create the PVCs and bind them to the `TaskRun`'s workspace.

**Substitute parameter, context and workspace and store in `TaskRun`'s status in [line 633](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L633):**

```go
 resources.ApplyParametersToWorkspaceBindings(rtr.TaskSpec, tr)
 workspaceVolumes := workspace.CreateVolumes(tr.Spec.Workspaces)

 ts, err := applyParamsContextsResultsAndWorkspaces(ctx, tr, rtr, workspaceVolumes)
 if err != nil {
  logger.Errorf(...)
  return err
 }
 tr.Status.TaskSpec = ts

```

**Create Pod for `TaskRun` in [line 648](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L648) and record a waring if scheduling fails due to resource limits:**

```go
 if pod == nil {
  pod, err = c.createPod(ctx, ts, tr, rtr, workspaceVolumes)
  // ...
 }
 if podconvert.IsPodExceedingNodeResources(pod) {
  recorder.Eventf(...)
 }
```

**Mark the `Pod` as ready if sidecars are ready in [line 661](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L661):**

```go
 if podconvert.SidecarsReady(pod.Status) {
  if err := podconvert.UpdateReady(ctx, c.KubeClientSet, *pod); err != nil {
   return err
  }
  //...
 }
```

**Update `TaskRun` status from `Pod` status in [line 671](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L671):**

```go
 tr.Status, err = podconvert.MakeTaskRunStatus(...)
 if err != nil {
  return err
 }
```

**Validate final results an log success message in [line 676](https://github.com/tektoncd/pipeline/blob/e97fed6c6442586bceccba8a7fad8dddbd78b3d6/pkg/reconciler/taskrun/taskrun.go#L676):**

```go
 if err := validateTaskRunResults(tr, rtr.TaskSpec); err != nil {
  tr.Status.MarkResourceFailed(v1.TaskRunReasonFailedValidation, err)
  return err
 }

 logger.Infof("Successfully reconciled taskrun...")
 return nil
```

And we are **done**.

### PipelineRun Controller

A brief diagram showing most, but not all, steps in the `PipelineRun` reconciler. Call chain of core reconcile methods:

```plaintext
FUNC: ReconcileKind
↓
setup and tracing
↓
read initial condition
↓
timeout check
↓
init on first start
↓
verification (chains)
↓
completion handling
↓
propagate pipeline name label
↓
cancellation check
↓
sync status with TaskRuns
↓
--> FUNC: updatePipelineRunStatusFromInformer
    ↓
    uses taskRunLister to get TaskRun
    ↓
    --> FUNC: updatePipelineRunStatusFromChildObjects
        ↓
        --> FUNC updatePipelineRunStatusFromChildRefs
            ↓
            updates PipelineRun's status with every TaskRun and CustomRun it owns
            ↓
            returns
            ↓
<------------
↓
main reconciliation logic
↓
--> FUNC: reconcile
    ↓
    setup, tracing, metrics
    ↓
    pending runs
    ↓
    pipeline resolution and validation
    ↓
    verification
    ↓
    DAG build
    ↓
    parameter, workspace and spec validation
    ↓
    parameter, workspace substitution
    ↓
    task state resolution
    ↓
    --> FUNC: resolvePipelineState
        ↓
        setup tracing
        ↓
        main loop: resolve each task
        ↓
        figure out TaskRun name
        ↓
        verification
        ↓
        prepare Task/CustomRun resolution (=fetch) function
        ↓
        resolve PipelineTask
        ↓
        --> FUNC: resources.ResolvePipelineTask
            ↓
            branch for Task / CustomTask
            ↓
            --> FUNC: setTaskRunsAndResolvedTask (getRun(runName) for CustomTask)
                ↓
                --> FUNC: resolveTask
                    ↓
                    put actual Task on ResolvedTask data structure
                    ↓
                    return
                    ↓
        <--------------
        ↓
        error handling
        ↓
        verification
        ↓
        append resolved Task state 
        ↓
        return
        ↓
    <----
    ↓
    build PipelineRunFacts for scheduling
    ↓
    Task/Param validations after resolution
    ↓
    CEL evaluation
    ↓
    cancellation, timeout handling
    ↓
    pre-flight checks: references valid, workspaces setup, affinity
    ↓
    scheduling next Task
    ↓
    --> FUNC: runNextSchedulableTask
        ↓
        get next executable task from DAG queue
        ↓
        validate result references
        ↓
        handle final tasks
        ↓
        propagate results to workspace bindings
        ↓
        main loop: actually schedule tasks
        ↓
        propagate results and artifacts
        ↓
        branch for Tasks and CustomTask
        ↓
        --> FUNC createTaskRuns (createCustomRuns for CustomTask)
            ↓
            validations
            ↓
            --> FUNC createTaskRun
                ↓
                create TaskRun instance
                ↓
                use client to create TaskRun CRD in cluster
                ↓
                return
                ↓
            <----
            ↓
            return
            ↓
        <----
        ↓
        return
        ↓
    <----
    ↓
    status calculation and finalization
    ↓
    return
    ↓
<----
↓
return requeue for timeouts
```
### Example

An example implementation of a CRD with a controller and a *reconciler* using the [kubebuilder](https://book.kubebuilder.io/getting-started) framework can be found in this repository in[samples/memcached-operator](../samples/memcached-operator/README.md).
