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
	"errors"

	appsv1 "k8s.io/api/apps/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cachev1alpha1 "example.com/m/v2/api/v1alpha1"
	"example.com/m/v2/internal/controller/infra"
)

var _ = Describe("Memcached Controller", func() {
	Context("When reconciling a resource (clean up deployment)", func() {
		resourceName, ctx, typeNamespacedName, memcached := baseSetup()

		BeforeEach(func() {
			createMemcachedCR(resourceName, ctx, typeNamespacedName, memcached)
		})

		AfterEach(func() {
			cleanUp(typeNamespacedName, true)
		})

		It("should set resource status to 'Unknown' during first reconciliation loop", func() {
			r := newReconciler()

			_, _ = reconcileOnce(ctx, r, typeNamespacedName, false)

			expectCondition(metav1.ConditionUnknown, "Reconciling", typeNamespacedName)
		})

		It("should set resource status to 'True' during second reconciliation loop", func() {
			r := newReconciler()

			By("Reconcile two times")
			_, _ = reconcileOnce(ctx, r, typeNamespacedName, false)
			_, _ = reconcileOnce(ctx, r, typeNamespacedName, false)

			By("Status 'True' after second reconciliation loop")
			expectCondition(metav1.ConditionTrue, "Reconciling", typeNamespacedName)
		})

		It("should set resource size back to 1 if it was changed", func() {
			r := newReconciler()

			By("Reconcile two times")
			_, _ = reconcileOnce(ctx, r, typeNamespacedName, false)
			_, _ = reconcileOnce(ctx, r, typeNamespacedName, false)

			By("Status 'True' after second reconciliation loop")
			expectCondition(metav1.ConditionTrue, "Reconciling", typeNamespacedName)

			By("Manually change deployment size to 2")
			resizeDeploymentTo(2, typeNamespacedName)

			By("Requeue after size was changed back to 1")
			result, err := reconcileOnce(ctx, r, typeNamespacedName, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())
		})

		It("should requeue with error if k8 client fails to update deployment replicas size", func() {
			expectedErr := errors.New("error updating the object")
			errMap := infra.StubErrors{"Update": {expectedErr}}

			r := newReconcilerNull(errMap, true)

			_, _ = reconcileOnce(ctx, r, typeNamespacedName, false)

			By("Status 'Unknown' after first reconciliation loop")
			expectCondition(metav1.ConditionUnknown, "Reconciling", typeNamespacedName)

			By("Manually change deployment size to 2")
			resizeDeploymentTo(2, typeNamespacedName)

			_, err := reconcileOnce(ctx, r, typeNamespacedName, true)
			Expect(err).To(MatchError(expectedErr))

			By("Status 'False' after second reconciliation loop")
			expectCondition(metav1.ConditionFalse, "Resizing", typeNamespacedName)
		})
	})

	Context("When reconciling a resource (no deployment clean up)", func() {
		resourceName, ctx, typeNamespacedName, memcached := baseSetup()

		BeforeEach(func() {
			createMemcachedCR(resourceName, ctx, typeNamespacedName, memcached)
		})

		AfterEach(func() {
			cleanUp(typeNamespacedName, false)
		})

		It("should set resource status to 'False' when setting controller reference for deployment fails", func() {
			r, errMsg := newReconcilerWithFailingSetter()

			By("Reconcile with error")
			_, err := reconcileOnce(ctx, r, typeNamespacedName, true)
			Expect(err.Error()).To(Equal(errMsg))

			By("Status 'False' after first reconciliation loop")
			expectCondition(metav1.ConditionFalse, "Reconciling", typeNamespacedName)
		})

		It("should requeue with error if k8 client fails to get the resource although it exists", func() {
			expectedErr := errors.New("error reading the object")
			errMap := infra.StubErrors{"Get": {expectedErr}}
			r := newReconcilerNull(errMap, false)

			_, err := reconcileOnce(ctx, r, typeNamespacedName, true)
			Expect(err).To(MatchError(expectedErr))

			expectNoDeployment(typeNamespacedName)
		})

		It("should requeue with error if k8 client fails to update memcached resource status", func() {
			expectedErr := errors.New("error updating resource status")
			errMap := infra.StubErrors{"StatusUpdate": {expectedErr}}
			r := newReconcilerNull(errMap, false)

			_, err := reconcileOnce(ctx, r, typeNamespacedName, true)
			Expect(err).To(MatchError(expectedErr))

			expectNoDeployment(typeNamespacedName)
		})

		It("should requeue with error if k8 client fails to get the memcached resource after status update", func() {
			expectedErr := errors.New("error reading the object")
			errMap := infra.StubErrors{"Get": {nil, expectedErr}}
			r := newReconcilerNull(errMap, false)

			_, err := reconcileOnce(ctx, r, typeNamespacedName, true)
			Expect(err).To(MatchError(expectedErr))

			expectNoDeployment(typeNamespacedName)
		})

		It("should requeue with error if k8 client fails to create deployment", func() {
			expectedErr := errors.New("error reading the object")
			errMap := infra.StubErrors{
				"Get":    {nil, nil, apierrors.NewNotFound(schema.GroupResource{}, "deployment not found")},
				"Create": {expectedErr},
			}
			r := newReconcilerNull(errMap, false)

			_, err := reconcileOnce(ctx, r, typeNamespacedName, true)
			Expect(err).To(MatchError(expectedErr))

			expectNoDeployment(typeNamespacedName)
		})

		It("should requeue with error if k8 client fails to get the deployment for other reasons than 'Not Found'", func() {
			expectedErr := apierrors.NewTimeoutError("getting the deployment timed out", 1)
			errMap := infra.StubErrors{
				"Get": {nil, nil, expectedErr},
			}
			r := newReconcilerNull(errMap, false)

			_, err := reconcileOnce(ctx, r, typeNamespacedName, true)
			Expect(err).To(MatchError(expectedErr))

			expectNoDeployment(typeNamespacedName)
		})
	})

	Context("When reconciling a non existing Memcached resource", func() {
		It("should not find the resource and stop the reconciliation loop", func() {
			_, ctx, typeNamespacedName, _ := baseSetup()
			r := newReconciler()

			_, _ = reconcileOnce(ctx, r, typeNamespacedName, false)

			err := k8sClient.Get(ctx, typeNamespacedName, &cachev1alpha1.Memcached{})
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})
})

func baseSetup() (string, context.Context, types.NamespacedName, *cachev1alpha1.Memcached) {
	const resourceName = "test-resource"
	ctx := context.Background()
	typeNamespacedName := types.NamespacedName{
		Name:      resourceName,
		Namespace: "default", // TODO(user):Modify as needed
	}
	memcached := &cachev1alpha1.Memcached{}

	return resourceName, ctx, typeNamespacedName, memcached
}

func createMemcachedCR(
	resourceName string,
	ctx context.Context,
	typeNamespacedName types.NamespacedName,
	memcached *cachev1alpha1.Memcached,
) {
	err := k8sClient.Get(ctx, typeNamespacedName, memcached)
	if err != nil && apierrors.IsNotFound(err) {
		resource := &cachev1alpha1.Memcached{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: "default",
			},
			// TODO(user): Specify other spec details if needed.
			Spec: cachev1alpha1.MemcachedSpec{
				Size: 1,
			},
		}
		Expect(k8sClient.Create(ctx, resource)).To(Succeed())
	}
}

func cleanUp(typeNamespacedName types.NamespacedName, withDeployment bool) {
	resource := &cachev1alpha1.Memcached{}
	err := k8sClient.Get(ctx, typeNamespacedName, resource)
	Expect(err).NotTo(HaveOccurred())

	By("Cleanup the specific resource instance Memcached")
	Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

	if !withDeployment {
		return
	}
	// INFO: apparently the deployment is not cleaned up in the test cluster when removing
	// the Memcached resource so you need to do it manually or other tests won't work
	dep := &appsv1.Deployment{}
	err = k8sClient.Get(ctx, typeNamespacedName, dep)
	Expect(err).NotTo(HaveOccurred())
	By("Cleanup the  Memcached deployment")
	Expect(k8sClient.Delete(ctx, dep)).To(Succeed())
}

func newReconciler() *MemcachedReconciler {
	return NewReconciler(k8sClient.Scheme(), k8sClient, ctrl.SetControllerReference)
}

func newReconcilerNull(errMap infra.StubErrors, withRealK8 bool) *MemcachedReconciler {
	return NewReconcilerNull(k8sClient.Scheme(), k8sClient, ctrl.SetControllerReference, errMap, withRealK8)
}

func newReconcilerWithFailingSetter() (*MemcachedReconciler, string) {
	errMsg := "Failed setting controller reference"
	ownerRefFor := func(_, _ metav1.Object, _ *runtime.Scheme, _ ...controllerutil.OwnerReferenceOption) error {
		return errors.New(errMsg)
	}

	return NewReconciler(k8sClient.Scheme(), k8sClient, ownerRefFor), errMsg
}

func reconcileOnce(c context.Context, r *MemcachedReconciler, t types.NamespacedName, expectFail bool) (ctrl.Result, error) {
	result, err := r.Reconcile(c, reconcile.Request{
		NamespacedName: t,
	})

	if expectFail {
		Expect(err).To(HaveOccurred())
	} else {
		Expect(err).NotTo(HaveOccurred())
	}

	return result, err
}

func expectNoDeployment(t types.NamespacedName) {
	By("No deployment was created")
	err := k8sClient.Get(ctx, t, &appsv1.Deployment{})
	Expect(err).To(HaveOccurred())
}

func resizeDeploymentTo(size int32, t types.NamespacedName) {
	// get
	dep := &appsv1.Deployment{}
	err := k8sClient.Get(ctx, t, dep)
	Expect(err).NotTo(HaveOccurred())

	// manually resize
	dep.Spec.Replicas = &size
	err = k8sClient.Update(ctx, dep)
	Expect(err).NotTo(HaveOccurred())

	// validate
	dep = &appsv1.Deployment{}
	err = k8sClient.Get(ctx, t, dep)
	Expect(err).NotTo(HaveOccurred())
	Expect(*dep.Spec.Replicas).To(Equal(int32(2)))
}

func expectCondition(status metav1.ConditionStatus, reason string, t types.NamespacedName) {
	updated := &cachev1alpha1.Memcached{}
	Expect(k8sClient.Get(ctx, t, updated)).To(Succeed())
	Expect(updated.Status.Conditions[0].Status).To(Equal(status))
	Expect(updated.Status.Conditions[0].Reason).To(Equal(reason))
}
