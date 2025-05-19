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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cachev1alpha1 "example.com/m/v2/api/v1alpha1"
)

var _ = Describe("Memcached Controller", func() {
	Context("When reconciling a resource", func() {
		resourceName, ctx, typeNamespacedName, memcached := baseSetup()

		BeforeEach(func() {
			By("creating the custom resource for the Kind Memcached")
			createMemcachedCR(resourceName, ctx, typeNamespacedName, memcached)
		})

		AfterEach(func() {
			cleanUp(typeNamespacedName, true)
		})

		It("should set resource status to 'Unknown' during first reconciliation loop", func() {
			controllerReconciler := &MemcachedReconciler{
				Client:                 k8sClient,
				Scheme:                 k8sClient.Scheme(),
				SetControllerReference: ctrl.SetControllerReference,
			}

			By("Reconcile the resource first time")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			updated := &cachev1alpha1.Memcached{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updated)).To(Succeed())
			Expect(updated.Status.Conditions[0].Status).To(Equal(metav1.ConditionUnknown))
			Expect(updated.Status.Conditions[0].Reason).To(Equal("Reconciling"))
		})

		It("should set resource status to 'True' during second reconciliation loop", func() {
			controllerReconciler := &MemcachedReconciler{
				Client:                 k8sClient,
				Scheme:                 k8sClient.Scheme(),
				SetControllerReference: ctrl.SetControllerReference,
			}

			By("Reconcile the resource first time")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Reconcile the resource second time")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Status 'True' after second reconciliation loop")
			updated := &cachev1alpha1.Memcached{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updated)).To(Succeed())
			Expect(updated.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(updated.Status.Conditions[0].Reason).To(Equal("Reconciling"))
		})
	})

	Context("When reconciling a resource and setting controller reference fails", func() {
		resourceName, ctx, typeNamespacedName, memcached := baseSetup()

		errMsg := "Failed setting controller reference"
		setControllerReferenceFake := func(_, _ metav1.Object, _ *runtime.Scheme, _ ...controllerutil.OwnerReferenceOption) error {
			return errors.New(errMsg)
		}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Memcached")
			createMemcachedCR(resourceName, ctx, typeNamespacedName, memcached)
		})

		AfterEach(func() {
			cleanUp(typeNamespacedName, false)
		})

		It("should set resource status to 'False' when setting controller reference fails", func() {
			controllerReconciler := &MemcachedReconciler{
				Client:                 k8sClient,
				Scheme:                 k8sClient.Scheme(),
				SetControllerReference: setControllerReferenceFake,
			}

			By("Reconcile the resource first time")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errMsg))

			By("Status 'False' after first reconciliation loop")
			updated := &cachev1alpha1.Memcached{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updated)).To(Succeed())
			Expect(updated.Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(updated.Status.Conditions[0].Reason).To(Equal("Reconciling"))
		})
	})

	Context("When reconciling a non existing resource", func() {
		It("should not find the resource and stop the reconciliation loop", func() {
			_, ctx, typeNamespacedName, _ := baseSetup()

			controllerReconciler := &MemcachedReconciler{
				Client:                 k8sClient,
				Scheme:                 k8sClient.Scheme(),
				SetControllerReference: ctrl.SetControllerReference,
			}

			By("Reconcile the resource first time")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedName, &cachev1alpha1.Memcached{})
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
