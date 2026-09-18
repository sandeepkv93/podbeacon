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

package v1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	telemetryv1alpha1 "github.com/sandeepkv93/podbeacon/api/v1alpha1"
)

const defaultNamespace = "default"

var _ = Describe("Pod Webhook", func() {
	Context("When defaulting a Pod", func() {
		ctx := context.Background()
		var defaulter *PodDefaulter

		BeforeEach(func() {
			defaulter = &PodDefaulter{Client: k8sClient}

			// Create a ready TelemetryProfile
			profile := &telemetryv1alpha1.TelemetryProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultNamespace,
					Namespace: defaultNamespace,
				},
				Spec: telemetryv1alpha1.TelemetryProfileSpec{
					Exporter: telemetryv1alpha1.ExporterSpec{
						Endpoint: "localhost:4317",
					},
				},
			}
			err := k8sClient.Create(ctx, profile)
			if err != nil {
				// if it already exists, just get it
				err = k8sClient.Get(ctx, types.NamespacedName{Name: defaultNamespace, Namespace: defaultNamespace}, profile)
				Expect(err).NotTo(HaveOccurred())
			}

			// Mark as ready
			profile.Status.Conditions = []metav1.Condition{
				{
					Type:               "Ready",
					Status:             "True",
					ObservedGeneration: profile.Generation,
					Reason:             "Ready",
					LastTransitionTime: metav1.Now(),
				},
			}
			profile.Status.ConfigMapName = "test-config"
			profile.Status.ConfigHash = "abcdef"
			err = k8sClient.Status().Update(ctx, profile)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should inject the sidecar if opted in", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: defaultNamespace,
					Annotations: map[string]string{
						"telemetry": "enable",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "busybox",
						},
					},
				},
			}

			err := defaulter.Default(ctx, pod)
			Expect(err).NotTo(HaveOccurred())

			// Check injection
			Expect(pod.Annotations["podbeacon.io/injected"]).To(Equal("true"))
			Expect(pod.Spec.InitContainers).To(HaveLen(1))
			Expect(pod.Spec.InitContainers[0].Name).To(Equal("podbeacon-collector"))
			Expect(string(*pod.Spec.InitContainers[0].RestartPolicy)).To(Equal("Always"))
		})

		It("should do nothing if not opted in", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-no-opt-in",
					Namespace: defaultNamespace,
				},
			}
			err := defaulter.Default(ctx, pod)
			Expect(err).NotTo(HaveOccurred())
			Expect(pod.Annotations["podbeacon.io/injected"]).To(BeEmpty())
		})
	})
})
