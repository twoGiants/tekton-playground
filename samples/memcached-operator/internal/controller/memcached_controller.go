/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "example.com/m/v2/api/v1alpha1"
)

const typeAvailableMemcached = "Available"

// MemcachedReconciler reconciles a Memcached object
type MemcachedReconciler struct {
	Scheme                 *runtime.Scheme
	SetControllerReference func(metav1.Object, metav1.Object, *runtime.Scheme, ...controllerutil.OwnerReferenceOption) error
	K8Cli                  K8CliWrapper
}

// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Memcached object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *MemcachedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Memcached instance (CR; remember a CR is like an instance of a CRD)
	// The purpose is to check if the Custom Resource for the Kind Memcached
	// is applied on the cluster. If not we return nil to stop the reconciliation.
	memcached := &cachev1alpha1.Memcached{}
	if err := r.K8Cli.Get(ctx, req.NamespacedName, memcached); err != nil {
		if apierrors.IsNotFound(err) {
			// If the CR is not found then it usually means that it was deleted or not created.
			// In this way, we will stop the reconciliation.
			log.Info("memcached resource not found. Ignoring since object must be deleted")
			return stop()
		}

		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get memcached")
		return requeueWith(err)
	}
	log.Info("memcached resource found")

	// Let's just set the status to Unknown when no status is available
	if len(memcached.Status.Conditions) == 0 {
		if err := r.updateStatus(ctx, memcached,
			metav1.ConditionUnknown, "Starting reconciliation"); err != nil {
			return requeueWith(err)
		}

		// Let's re-fetch the memcached CR after updating the status
		// so that we have the latest state of the resource on the cluster and we will avoid
		// raising the error "the object has been modified, please apply
		// your changes to the latest version and try again" which would re-trigger the reconciliation
		// if we try to update it again in the following operations
		if err := r.K8Cli.Get(ctx, req.NamespacedName, memcached); err != nil {
			log.Error(err, "Failed to re-fetch memcached")
			return requeueWith(err)
		}
		log.Info("no status available, set to Unknown")
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err := r.K8Cli.Get(ctx, types.NamespacedName{Name: memcached.Name, Namespace: memcached.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new deployment
		dep, err := r.deploymentForMemcached(memcached)
		if err != nil {
			log.Error(err, "Failed to define new Deployment resource for Memcached")

			// The following implementation will update the status
			if err := r.updateStatus(ctx, memcached,
				metav1.ConditionFalse,
				fmt.Sprintf("Failed to create Deployment for the custom resource (%s): (%s)", memcached.Name, err),
			); err != nil {
				return requeueWith(err)
			}

			return requeueWith(err)
		}

		// Create the deployment in the cluster
		log.Info("Creating a new Deployment",
			"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		if err := r.K8Cli.Create(ctx, dep); err != nil {
			log.Error(err, "Failed to create new Deployment",
				"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)

			return requeueWith(err)
		}

		// Deployment created successfully
		// We will requeue the reconciliation so that we can ensure the state
		// and move forward for the next operations
		return requeueAfterMinute()
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		// Let's return the error for the reconciliation to be re-triggered again
		return requeueWith(err)
	}

	// The CRD API defines that the Memcached type have a MemcachedSpec.Size field
	// to set the quantity of DEployment instances to the desired state on the cluster.
	// Therefore, the following code will ensure the Deployment size is the same as defined
	// via the Size spec of the Custom Resource which we are reconciling.
	log.Info("reconciling size",
		"Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
	size := memcached.Spec.Size
	if *found.Spec.Replicas != size {
		log.Info(fmt.Sprintf("found diverging size (%d), changing back to (%d)", *found.Spec.Replicas, size),
			"Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
		found.Spec.Replicas = &size
		if err := r.K8Cli.Update(ctx, found); err != nil {
			log.Error(err, "Failed to update Deployment",
				"Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)

			// Re-fetch the memcached Custom Resource before updating the status
			// so that we have the latest state of the resource on the cluster and we will avoid
			// raising the error "the object has been modified, please apply your changes to the
			// latest version and try again" which would re-trigger the reconciliation
			if err := r.K8Cli.Get(ctx, req.NamespacedName, memcached); err != nil {
				log.Error(err, "Failed to re-fetch memcached")
				return requeueWith(err)
			}

			// The following implementation will update the status
			meta.SetStatusCondition(
				&memcached.Status.Conditions,
				metav1.Condition{
					Type:    typeAvailableMemcached,
					Status:  metav1.ConditionFalse,
					Reason:  "Resizing",
					Message: fmt.Sprintf("Failed to update the size for the custom resource (%s): (%s)", memcached.Name, err),
				},
			)
			if err := r.K8Cli.StatusUpdate(ctx, memcached); err != nil {
				log.Error(err, "Failed to update Memcached status")
				return requeueWith(err)
			}

			return requeueWith(err)
		}

		// Now, that we update the size we want to requeue the reconciliation
		// so that we can ensure that we have the latest state of the resource before
		// update. Also, it will help ensure the desired state on the cluster
		return requeue()
	} else {
		log.Info("all good, no drift in size found",
			"Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
	}

	// The following implementation will update the status
	meta.SetStatusCondition(
		&memcached.Status.Conditions,
		metav1.Condition{
			Type:    typeAvailableMemcached,
			Status:  metav1.ConditionTrue,
			Reason:  "Reconciling",
			Message: fmt.Sprintf("Deployment for custom resource (%s) with %d replicas created successfully", memcached.Name, size),
		},
	)
	if err := r.K8Cli.StatusUpdate(ctx, memcached); err != nil {
		log.Error(err, "Failed to update Memcached status")
		return requeueWith(err)
	}

	return stop()
}

func (r *MemcachedReconciler) updateStatus(
	ctx context.Context,
	memcached *cachev1alpha1.Memcached,
	status metav1.ConditionStatus,
	message string,
) error {
	log := logf.FromContext(ctx)

	meta.SetStatusCondition(
		&memcached.Status.Conditions,
		metav1.Condition{
			Type:    typeAvailableMemcached,
			Status:  status,
			Reason:  "Reconciling",
			Message: message,
		},
	)
	if err := r.K8Cli.StatusUpdate(ctx, memcached); err != nil {
		log.Error(err, "Failed to update Memcached status")
		return err
	}

	return nil
}

func (r *MemcachedReconciler) deploymentForMemcached(memcached *cachev1alpha1.Memcached) (*appsv1.Deployment, error) {
	replicas := memcached.Spec.Size
	image := "memcached:1.6.26-alpine3.19"

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      memcached.Name,
			Namespace: memcached.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "project"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": "project"},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptr.To(true),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{{
						Image:           image,
						Name:            "memcached",
						ImagePullPolicy: corev1.PullIfNotPresent,
						// Ensure restrictive context for the container
						// More info: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted
						SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             ptr.To(true),
							RunAsUser:                ptr.To(int64(1001)),
							AllowPrivilegeEscalation: ptr.To(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 11211,
							Name:          "memcached",
						}},
						Command: []string{"memcached", "--memory-limit=64", "-o", "modern", "-v"},
					}},
				},
			},
		},
	}

	// Set the ownerRef for the Deployment. Important so that reconciliation is triggered when the
	// Deployment of our Memcached Custom Resource is changed and when the Memcached Custom Resource
	// is deleted all resources owned by it are also automatically deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := r.SetControllerReference(memcached, dep, r.Scheme); err != nil {
		return nil, err
	}
	return dep, nil
}

// SetupWithManager sets up the controller with the Manager.
// The deployment is also watched to ensure its desired state in the cluster.
func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Watch the Memcached Custom Resource and trigger reconciliation whenever it
		// is created, updated, or deleted.
		For(&cachev1alpha1.Memcached{}).
		// Watch the Deployment managed by the Memcached controller. If any changes occur to the
		// Deployment owned and managed by this controller, it will trigger reconciliation, ensuring
		// that the cluster state aligns with the desired state.
		Owns(&appsv1.Deployment{}).
		Named("memcached").
		Complete(r)
}

func stop() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func requeueWith(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}

func requeue() (ctrl.Result, error) {
	return ctrl.Result{Requeue: true}, nil
}

func requeueAfterMinute() (ctrl.Result, error) {
	return ctrl.Result{RequeueAfter: time.Minute}, nil
}
