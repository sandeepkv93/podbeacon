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
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	telemetryv1alpha1 "github.com/sandeepkv93/podbeacon/api/v1alpha1"
)

// TelemetryProfileReconciler reconciles a TelemetryProfile object
type TelemetryProfileReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=telemetry.podbeacon.io,resources=telemetryprofiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=telemetry.podbeacon.io,resources=telemetryprofiles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=telemetry.podbeacon.io,resources=telemetryprofiles/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *TelemetryProfileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the TelemetryProfile instance
	profile := &telemetryv1alpha1.TelemetryProfile{}
	if err := r.Get(ctx, req.NamespacedName, profile); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Initialize status conditions if they don't exist
	if profile.Status.Conditions == nil {
		profile.Status.Conditions = []metav1.Condition{}
	}

	// Check Secrets
	var headerKeys []string
	if profile.Spec.Exporter.HeadersSecretRef != nil {
		secretName := profile.Spec.Exporter.HeadersSecretRef.Name
		secret := &corev1.Secret{}
		err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: req.Namespace}, secret)
		if err != nil {
			log.Error(err, "Failed to get headers Secret", "Secret.Name", secretName)
			return r.updateStatus(ctx, profile, false, "MissingSecret", fmt.Sprintf("Headers Secret %s not found", secretName))
		}
		for k := range secret.Data {
			headerKeys = append(headerKeys, k)
		}
	}

	if profile.Spec.Exporter.TLS != nil && profile.Spec.Exporter.TLS.CASecretRef != nil {
		secretName := profile.Spec.Exporter.TLS.CASecretRef.Name
		secret := &corev1.Secret{}
		err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: req.Namespace}, secret)
		if err != nil {
			log.Error(err, "Failed to get CA Secret", "Secret.Name", secretName)
			return r.updateStatus(ctx, profile, false, "MissingSecret", fmt.Sprintf("CA Secret %s not found", secretName))
		}
		// Verify key exists
		key := profile.Spec.Exporter.TLS.CASecretRef.Key
		if key != "" {
			if _, ok := secret.Data[key]; !ok {
				return r.updateStatus(ctx, profile, false, "MissingSecret", fmt.Sprintf("Key %s not found in CA Secret %s", key, secretName))
			}
		}
	}

	// Generate ConfigMap content
	configStr, hash, err := GenerateConfig(profile, headerKeys)
	if err != nil {
		log.Error(err, "Failed to generate config")
		return r.updateStatus(ctx, profile, false, "InvalidConfiguration", err.Error())
	}

	cmName := fmt.Sprintf("podbeacon-cfg-%s-%s", string(profile.UID)[:8], hash)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: req.Namespace,
			Labels: map[string]string{
				"podbeacon.io/profile":     profile.Name,
				"podbeacon.io/config-hash": hash,
				"podbeacon.io/profile-uid": string(profile.UID),
			},
		},
		//nolint:modernize
		Immutable: ptr.To(true),
		Data: map[string]string{
			"relay.yaml": configStr,
		},
	}

	err = r.Create(ctx, cm)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			// Verify it matches
			existing := &corev1.ConfigMap{}
			if err := r.Get(ctx, types.NamespacedName{Name: cmName, Namespace: req.Namespace}, existing); err != nil {
				return r.updateStatus(ctx, profile, false, "ConfigurationPending", "Failed to retrieve existing ConfigMap")
			}
			if existing.Data["relay.yaml"] != configStr {
				return r.updateStatus(ctx, profile, false, "ConfigurationConflict", "Existing ConfigMap data mismatch")
			}
		} else {
			return r.updateStatus(ctx, profile, false, "ConfigurationPending", "Failed to create ConfigMap: "+err.Error())
		}
	} else {
		log.Info("ConfigMap materialized", "ConfigMap.Name", cmName)
	}

	// Update Status to Ready
	profile.Status.ConfigHash = hash
	profile.Status.ConfigMapName = cmName
	profile.Status.ObservedGeneration = profile.Generation

	return r.updateStatus(ctx, profile, true, "Ready", "Configuration materialized successfully")
}

func (r *TelemetryProfileReconciler) updateStatus(ctx context.Context, profile *telemetryv1alpha1.TelemetryProfile, isReady bool, reason, message string) (ctrl.Result, error) {
	status := metav1.ConditionFalse
	if isReady {
		status = metav1.ConditionTrue
	}

	condition := metav1.Condition{
		Type:               "Ready",
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: profile.Generation,
	}

	// Check if condition actually changed
	existing := meta.FindStatusCondition(profile.Status.Conditions, "Ready")
	changed := existing == nil || existing.Status != status || existing.Reason != reason || existing.Message != message || existing.ObservedGeneration != profile.Generation

	if changed {
		meta.SetStatusCondition(&profile.Status.Conditions, condition)
		err := r.Status().Update(ctx, profile)
		if err != nil {
			return ctrl.Result{RequeueAfter: time.Second}, err
		}
	}

	if !isReady && reason != "MissingSecret" && reason != "InvalidConfiguration" {
		// Return error for backoff on materialization failure, etc.
		return ctrl.Result{}, fmt.Errorf("reconciliation failed: %s", message)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TelemetryProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Map Secret to profiles
	mapSecretToProfile := func(ctx context.Context, obj client.Object) []reconcile.Request {
		secret := obj.(*corev1.Secret)
		var profiles telemetryv1alpha1.TelemetryProfileList
		if err := r.List(ctx, &profiles, client.InNamespace(secret.Namespace)); err != nil {
			return nil
		}
		var reqs []reconcile.Request
		for _, p := range profiles.Items {
			if (p.Spec.Exporter.HeadersSecretRef != nil && p.Spec.Exporter.HeadersSecretRef.Name == secret.Name) ||
				(p.Spec.Exporter.TLS != nil && p.Spec.Exporter.TLS.CASecretRef != nil && p.Spec.Exporter.TLS.CASecretRef.Name == secret.Name) {
				reqs = append(reqs, reconcile.Request{
					NamespacedName: types.NamespacedName{Name: p.Name, Namespace: p.Namespace},
				})
			}
		}
		return reqs
	}

	// Map ConfigMap to profiles
	mapCMToProfile := func(ctx context.Context, obj client.Object) []reconcile.Request {
		cm := obj.(*corev1.ConfigMap)
		if profileName, ok := cm.Labels["podbeacon.io/profile"]; ok {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{Name: profileName, Namespace: cm.Namespace}},
			}
		}
		return nil
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&telemetryv1alpha1.TelemetryProfile{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(mapSecretToProfile)).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(mapCMToProfile)).
		Named("telemetryprofile").
		Complete(r)
}
