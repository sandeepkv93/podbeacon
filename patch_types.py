import re

with open('api/v1alpha1/telemetryprofile_types.go', 'r') as f:
    content = f.read()

# Replace QueueSize default and validation
# Currently:
# 	// +kubebuilder:default=256
# 	// +kubebuilder:validation:Minimum=1
# 	// +kubebuilder:validation:Maximum=4096
# 	QueueSize *int32 `json:"queueSize,omitempty"`

# Duration validation:
# RetryMaxElapsedTime
content = content.replace(
    '	// +kubebuilder:default="30s"\n	RetryMaxElapsedTime *metav1.Duration `json:"retryMaxElapsedTime,omitempty"`',
    '	// +kubebuilder:default="30s"\n\t// +kubebuilder:validation:Type=string\n\t// +kubebuilder:validation:Format=duration\n\tRetryMaxElapsedTime *metav1.Duration `json:"retryMaxElapsedTime,omitempty"`'
)

# BatchSpec timeout
content = content.replace(
    '	// +kubebuilder:default="5s"\n	Timeout *metav1.Duration `json:"timeout,omitempty"`',
    '	// +kubebuilder:default="5s"\n\t// +kubebuilder:validation:Type=string\n\t// +kubebuilder:validation:Format=duration\n\tTimeout *metav1.Duration `json:"timeout,omitempty"`'
)

# Add XValidation for requests <= limits
# small memory limits: minimum resource validation
# visible defaults for signals/resources
# The CRD has bounds for a few integers but no duration bounds, no requests-not-greater-than-limits validation, no minimum resource validation, and no visible defaults for signals/resources.

# resources validation
resources_marker = """	// +optional
	// +kubebuilder:validation:XValidation:rule="!has(self.requests) || !has(self.limits) || !has(self.requests.cpu) || !has(self.limits.cpu) || string(self.requests.cpu) == '' || string(self.limits.cpu) == '' || quantity(self.requests.cpu) <= quantity(self.limits.cpu)",message="CPU requests must be less than or equal to limits"
	// +kubebuilder:validation:XValidation:rule="!has(self.requests) || !has(self.limits) || !has(self.requests.memory) || !has(self.limits.memory) || string(self.requests.memory) == '' || string(self.limits.memory) == '' || quantity(self.requests.memory) <= quantity(self.limits.memory)",message="Memory requests must be less than or equal to limits"
	// +kubebuilder:validation:XValidation:rule="!has(self.limits) || !has(self.limits.memory) || string(self.limits.memory) == '' || quantity(self.limits.memory) >= quantity('64Mi')",message="Memory limit must be at least 64Mi"
	// +kubebuilder:default={limits: {cpu: "100m", memory: "128Mi"}, requests: {cpu: "10m", memory: "64Mi"}}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`"""

content = content.replace(
    '	// +optional\n	Resources corev1.ResourceRequirements `json:"resources,omitempty"`',
    resources_marker
)

signals_marker = """	// +optional
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=3
	// +listType=set
	// +kubebuilder:default={traces,metrics}
	Signals []SignalType `json:"signals,omitempty"`"""

content = content.replace(
    '	// +optional\n	// +kubebuilder:validation:MinItems=1\n	// +kubebuilder:validation:MaxItems=3\n	// +listType=set\n	Signals []SignalType `json:"signals,omitempty"`',
    signals_marker
)


with open('api/v1alpha1/telemetryprofile_types.go', 'w') as f:
    f.write(content)
