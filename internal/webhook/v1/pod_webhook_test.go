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

//nolint:goconst,ineffassign

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
		var profile *telemetryv1alpha1.TelemetryProfile
		var cm *corev1.ConfigMap

		BeforeEach(func() {
			defaulter = &PodDefaulter{Client: k8sClient}

			profile = &telemetryv1alpha1.TelemetryProfile{
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
				err = k8sClient.Get(ctx, types.NamespacedName{Name: defaultNamespace, Namespace: defaultNamespace}, profile)
				Expect(err).NotTo(HaveOccurred())
			}

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

			// Create ConfigMap
			t := true
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: defaultNamespace,
					Labels: map[string]string{
						AnnotationConfigHash: "abcdef",
						AnnotationProfileUID: string(profile.UID),
					},
				},
				Immutable: &t,
				Data: map[string]string{
					"relay.yaml": "some-config",
				},
			}
			_ = k8sClient.Delete(ctx, cm)
			err = k8sClient.Create(ctx, cm)
			Expect(err).NotTo(HaveOccurred())
			if err != nil {
				_ = k8sClient.Delete(ctx, cm)
				err = k8sClient.Create(ctx, cm)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		AfterEach(func() {
			cmToDelete := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test-config", Namespace: defaultNamespace}}
			_ = k8sClient.Delete(ctx, cmToDelete)
			_ = k8sClient.Delete(ctx, profile)
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
					Containers: []corev1.Container{{Name: "app", Image: "busybox"}},
				},
			}

			err := defaulter.Default(ctx, pod)
			Expect(err).NotTo(HaveOccurred())

			Expect(pod.Annotations["podbeacon.io/injected"]).To(Equal("true"))
			Expect(pod.Spec.InitContainers).To(HaveLen(1))
			c := pod.Spec.InitContainers[0]
			Expect(c.Name).To(Equal("podbeacon-collector"))
			Expect(string(*c.RestartPolicy)).To(Equal("Always"))

			// Check for finding 11: Liveness probe
			Expect(c.LivenessProbe).NotTo(BeNil())
			Expect(c.LivenessProbe.HTTPGet.Path).To(Equal("/"))

			// Check for finding 11: Downward API
			envNames := make(map[string]bool)
			for _, e := range c.Env {
				envNames[e.Name] = true
			}
			Expect(envNames["POD_NAME"]).To(BeTrue())
			Expect(envNames["POD_NAMESPACE"]).To(BeTrue())
			Expect(envNames["NODE_NAME"]).To(BeTrue())
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

		It("should fail if Windows pod", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-windows",
					Namespace:   defaultNamespace,
					Annotations: map[string]string{"telemetry": "enable"},
				},
				Spec: corev1.PodSpec{
					OS: &corev1.PodOS{Name: corev1.Windows},
				},
			}
			err := defaulter.Default(ctx, pod)
			Expect(err).To(MatchError("windows pods are not supported"))
		})

		It("should fail if foreign OTel annotation present", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-foreign",
					Namespace: defaultNamespace,
					Annotations: map[string]string{
						"telemetry":                       "enable",
						"sidecar.opentelemetry.io/inject": "true",
					},
				},
			}
			err := defaulter.Default(ctx, pod)
			Expect(err).To(MatchError("foreign OpenTelemetry injection annotation detected"))
		})

		It("should fail if podbeacon-config volume exists", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-vol-conflict",
					Namespace:   defaultNamespace,
					Annotations: map[string]string{"telemetry": "enable"},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{Name: "podbeacon-config"}},
				},
			}
			err := defaulter.Default(ctx, pod)
			Expect(err).To(MatchError("volume podbeacon-config already exists"))
		})

		It("should fail if ConfigMap is missing relay.yaml", func() {
			err := k8sClient.Delete(ctx, cm)
			Expect(err).NotTo(HaveOccurred())

			t := true
			newCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: defaultNamespace,
					Labels: map[string]string{
						AnnotationConfigHash: "abcdef",
						AnnotationProfileUID: string(profile.UID),
					},
				},
				Immutable: &t,
				Data: map[string]string{
					"other.yaml": "some-config",
				},
			}
			err = k8sClient.Create(ctx, newCm)
			Expect(err).NotTo(HaveOccurred())

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-cm",
					Namespace:   defaultNamespace,
					Annotations: map[string]string{"telemetry": "enable"},
				},
			}
			err = defaulter.Default(ctx, pod)
			Expect(err).To(MatchError(ContainSubstring("missing relay.yaml")))
		})

		It("should fail if spoofed marker but incomplete injection", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-spoofed",
					Namespace: defaultNamespace,
					Annotations: map[string]string{
						"telemetry":          "enable",
						AnnotationInjected:   "true",
						AnnotationProfileUID: string(profile.UID),
						AnnotationConfigHash: "abcdef",
					},
				},
				Spec: corev1.PodSpec{},
			}
			err := defaulter.Default(ctx, pod)
			Expect(err).To(MatchError("injected container not found"))
		})

		It("repeated admission should succeed without error", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: defaultNamespace,
					Annotations: map[string]string{
						"telemetry": "enable",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "busybox"}},
				},
			}

			// First injection
			err := defaulter.Default(ctx, pod)
			Expect(err).NotTo(HaveOccurred())

			// Repeated admission
			err = defaulter.Default(ctx, pod)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
