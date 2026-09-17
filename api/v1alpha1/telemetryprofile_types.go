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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SignalType defines the telemetry signal
// +kubebuilder:validation:Enum=traces;metrics;logs
type SignalType string

const (
	SignalTraces  SignalType = "traces"
	SignalMetrics SignalType = "metrics"
	SignalLogs    SignalType = "logs"
)

// ExporterTLS defines the TLS configuration for the exporter
type ExporterTLS struct {
	// Insecure disables TLS verification when set to true. Default is false.
	// +optional
	Insecure bool `json:"insecure,omitempty"`

	// CASecretRef points to a Secret in the same namespace containing the CA cert
	// +optional
	CASecretRef *corev1.SecretKeySelector `json:"caSecretRef,omitempty"`
}

// ExporterSpec defines the destination for telemetry
type ExporterSpec struct {
	// Endpoint is the required OTLP/gRPC host:port
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9.-]+:\d+$`
	Endpoint string `json:"endpoint"`

	// TLS configuration for the exporter
	// +optional
	TLS *ExporterTLS `json:"tls,omitempty"`

	// HeadersSecretRef points to a Secret whose keys map to outbound header names.
	// +optional
	HeadersSecretRef *corev1.LocalObjectReference `json:"headersSecretRef,omitempty"`

	// QueueSize is the number of batches to queue. Default 256, max 4096.
	// +optional
	// +kubebuilder:default=256
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=4096
	QueueSize *int32 `json:"queueSize,omitempty"`

	// RetryMaxElapsedTime is the maximum time spent retrying a batch. Default 30s.
	// +optional
	// +kubebuilder:default="30s"
	RetryMaxElapsedTime *metav1.Duration `json:"retryMaxElapsedTime,omitempty"`
}

// BatchSpec defines the batch processor configuration
type BatchSpec struct {
	// Timeout is the time to wait before flushing a batch. Default 5s.
	// +optional
	// +kubebuilder:default="5s"
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// SendBatchSize is the number of spans/metrics/logs to batch. Default 512.
	// +optional
	// +kubebuilder:default=512
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=8192
	SendBatchSize *int32 `json:"sendBatchSize,omitempty"`
}

// TelemetryProfileSpec defines the desired state of TelemetryProfile
type TelemetryProfileSpec struct {
	// Exporter configures the OTLP destination
	// +required
	Exporter ExporterSpec `json:"exporter"`

	// Signals is a unique subset of traces, metrics, logs
	// +optional
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=3
	// +listType=set
	Signals []SignalType `json:"signals,omitempty"`

	// Resources configures the collector sidecar resources
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Batch configures the batch processor
	// +optional
	Batch *BatchSpec `json:"batch,omitempty"`
}

// TelemetryProfileStatus defines the observed state of TelemetryProfile
type TelemetryProfileStatus struct {
	// ObservedGeneration is the latest generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ConfigHash is the hash of the inputs used to generate the current ConfigMap.
	// +optional
	ConfigHash string `json:"configHash,omitempty"`

	// ConfigMapName is the name of the generated ConfigMap containing the collector config.
	// +optional
	ConfigMapName string `json:"configMapName,omitempty"`

	// Conditions represent the current state of the TelemetryProfile resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="Is the profile ready"
// +kubebuilder:printcolumn:name="ConfigMap",type="string",JSONPath=".status.configMapName",description="Generated ConfigMap name"

// TelemetryProfile is the Schema for the telemetryprofiles API
type TelemetryProfile struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec TelemetryProfileSpec `json:"spec"`

	// +optional
	Status TelemetryProfileStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// TelemetryProfileList contains a list of TelemetryProfile
type TelemetryProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []TelemetryProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TelemetryProfile{}, &TelemetryProfileList{})
}
