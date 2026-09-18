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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	telemetryv1alpha1 "github.com/sandeepkv93/podbeacon/api/v1alpha1"
)

var podlog = logf.Log.WithName("pod-webhook")

const (
	AnnotationTelemetryOptIn = "telemetry"
	AnnotationProfile        = "podbeacon.io/profile"
	AnnotationInjected       = "podbeacon.io/injected"
	AnnotationProfileUID     = "podbeacon.io/profile-uid"
	AnnotationConfigHash     = "podbeacon.io/config-hash"

	CollectorContainerName = "podbeacon-collector"
	CollectorImage         = "otel/opentelemetry-collector:0.120.0"
)

func SetupPodWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &corev1.Pod{}).
		WithDefaulter(&PodDefaulter{Client: mgr.GetClient()}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=fail,sideEffects=None,reinvocationPolicy=IfNeeded,groups="",resources=pods,verbs=create,versions=v1,name=mpod.telemetry.podbeacon.io,admissionReviewVersions=v1
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=telemetry.podbeacon.io,resources=telemetryprofiles,verbs=get;list;watch

type PodDefaulter struct {
	Client client.Client
}

func (d *PodDefaulter) Default(ctx context.Context, obj *corev1.Pod) error {
	// Skip if it doesn't have the exact opt-in annotation
	if obj.Annotations == nil || obj.Annotations[AnnotationTelemetryOptIn] != "enable" {
		return nil
	}

	// Skip if host network (INJ-08)
	if obj.Spec.HostNetwork {
		return fmt.Errorf("PodBeacon cannot inject into hostNetwork Pods")
	}

	// Skip if already injected with our marker (INJ-07)
	if obj.Annotations[AnnotationInjected] == "true" {
		return nil
	}

	// Check for conflicting containers
	for _, c := range obj.Spec.InitContainers {
		if c.Name == CollectorContainerName {
			return fmt.Errorf("container name conflict: %s already exists", CollectorContainerName)
		}
	}
	for _, c := range obj.Spec.Containers {
		if c.Name == CollectorContainerName {
			return fmt.Errorf("container name conflict: %s already exists", CollectorContainerName)
		}
	}

	profileName := obj.Annotations[AnnotationProfile]
	if profileName == "" {
		profileName = "default"
	}

	profile := &telemetryv1alpha1.TelemetryProfile{}
	err := d.Client.Get(ctx, types.NamespacedName{Name: profileName, Namespace: obj.Namespace}, profile)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("required TelemetryProfile '%s' not found", profileName)
		}
		return err
	}

	// Check readiness (ADM-04)
	isReady := false
	for _, c := range profile.Status.Conditions {
		if c.Type == "Ready" && c.Status == "True" && c.ObservedGeneration == profile.Generation {
			isReady = true
			break
		}
	}
	if !isReady || profile.Status.ConfigMapName == "" {
		return fmt.Errorf("TelemetryProfile '%s' is not ready or missing configuration", profileName)
	}

	// Create injection
	restartPolicy := corev1.ContainerRestartPolicyAlways

	// Default resources
	reqCPU := resource.MustParse("50m")
	reqMem := resource.MustParse("64Mi")
	limCPU := resource.MustParse("200m")
	limMem := resource.MustParse("128Mi")

	if profile.Spec.Resources.Requests != nil {
		if q, ok := profile.Spec.Resources.Requests[corev1.ResourceCPU]; ok {
			reqCPU = q
		}
		if q, ok := profile.Spec.Resources.Requests[corev1.ResourceMemory]; ok {
			reqMem = q
		}
	}
	if profile.Spec.Resources.Limits != nil {
		if q, ok := profile.Spec.Resources.Limits[corev1.ResourceCPU]; ok {
			limCPU = q
		}
		if q, ok := profile.Spec.Resources.Limits[corev1.ResourceMemory]; ok {
			limMem = q
		}
	}

	sidecar := corev1.Container{
		Name:          CollectorContainerName,
		Image:         CollectorImage,
		RestartPolicy: &restartPolicy,
		Args: []string{
			"--config=/conf/relay.yaml",
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    reqCPU,
				corev1.ResourceMemory: reqMem,
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    limCPU,
				corev1.ResourceMemory: limMem,
			},
		},
		StartupProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/",
					Port: intstr.FromInt(13133),
				},
			},
			FailureThreshold: 3,
			PeriodSeconds:    10,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "podbeacon-config",
				MountPath: "/conf",
				ReadOnly:  true,
			},
		},
	}

	// Secret headers mount
	if profile.Spec.Exporter.HeadersSecretRef != nil {
		secretName := profile.Spec.Exporter.HeadersSecretRef.Name
		secret := &corev1.Secret{}
		err := d.Client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: obj.Namespace}, secret)
		if err != nil {
			return fmt.Errorf("headers secret '%s' not found for profile", secretName)
		}
		for k := range secret.Data {
			sidecar.Env = append(sidecar.Env, corev1.EnvVar{
				Name: k,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
						Key:                  k,
					},
				},
			})
		}
	}

	obj.Spec.InitContainers = append(obj.Spec.InitContainers, sidecar)
	obj.Spec.Volumes = append(obj.Spec.Volumes, corev1.Volume{
		Name: "podbeacon-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: profile.Status.ConfigMapName,
				},
			},
		},
	})

	if obj.Annotations == nil {
		obj.Annotations = make(map[string]string)
	}
	obj.Annotations[AnnotationInjected] = "true"
	obj.Annotations[AnnotationProfileUID] = string(profile.UID)
	obj.Annotations[AnnotationConfigHash] = profile.Status.ConfigHash

	podlog.Info("Successfully injected PodBeacon collector sidecar", "namespace", obj.Namespace, "name", obj.Name, "profile", profileName)

	return nil
}
