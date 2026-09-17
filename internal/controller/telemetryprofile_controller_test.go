/*
Copyright 2026.

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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	telemetryv1alpha1 "github.com/sandeepkv93/podbeacon/api/v1alpha1"
)

var _ = Describe("TelemetryProfile Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName      = "test-resource"
			resourceNamespace = "default"
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNamespace,
		}

		BeforeEach(func() {
			By("creating the custom resource for the Kind TelemetryProfile")
			resource := &telemetryv1alpha1.TelemetryProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: resourceNamespace,
				},
				Spec: telemetryv1alpha1.TelemetryProfileSpec{
					Exporter: telemetryv1alpha1.ExporterSpec{
						Endpoint: "10.0.0.1:4317",
						TLS: &telemetryv1alpha1.ExporterTLS{
							Insecure: true,
						},
					},
					Signals: []telemetryv1alpha1.SignalType{
						telemetryv1alpha1.SignalTraces,
					},
				},
			}
			err := k8sClient.Create(ctx, resource)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			resource := &telemetryv1alpha1.TelemetryProfile{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should successfully reconcile the resource and generate a ConfigMap", func() {
			By("Reconciling the created resource")
			controllerReconciler := &TelemetryProfileReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the updated status")
			resource := &telemetryv1alpha1.TelemetryProfile{}
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, resource)
			}, time.Second*5, time.Millisecond*500).Should(Succeed())

			Expect(resource.Status.ConfigMapName).ToNot(BeEmpty())
			Expect(resource.Status.ConfigHash).ToNot(BeEmpty())

			var readyCond *metav1.Condition
			for _, c := range resource.Status.Conditions {
				if c.Type == "Ready" {
					cCopy := c
					readyCond = &cCopy
					break
				}
			}
			Expect(readyCond).ToNot(BeNil())
			Expect(readyCond.Status).To(Equal(metav1.ConditionTrue))

			By("Checking the generated ConfigMap")
			cm := &corev1.ConfigMap{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      resource.Status.ConfigMapName,
				Namespace: resourceNamespace,
			}, cm)
			Expect(err).NotTo(HaveOccurred())
			Expect(cm.Data).To(HaveKey("relay.yaml"))
			Expect(cm.Data["relay.yaml"]).To(ContainSubstring("10.0.0.1:4317"))
		})
	})
})
