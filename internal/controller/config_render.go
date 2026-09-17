package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	telemetryv1alpha1 "github.com/sandeepkv93/podbeacon/api/v1alpha1"
)

// OTelConfig represents the structure of the collector config
type OTelConfig struct {
	Receivers  Receivers  `yaml:"receivers"`
	Processors Processors `yaml:"processors"`
	Exporters  Exporters  `yaml:"exporters"`
	Service    Service    `yaml:"service"`
}

type Receivers struct {
	OTLP OTLPReceiver `yaml:"otlp"`
}

type OTLPReceiver struct {
	Protocols Protocols `yaml:"protocols"`
}

type Protocols struct {
	GRPC Endpoint `yaml:"grpc"`
	HTTP Endpoint `yaml:"http"`
}

type Endpoint struct {
	Endpoint string `yaml:"endpoint"`
}

type Processors struct {
	MemoryLimiter MemoryLimiter `yaml:"memory_limiter"`
	Batch         Batch         `yaml:"batch"`
}

type MemoryLimiter struct {
	CheckInterval string `yaml:"check_interval"`
	LimitMiB      int32  `yaml:"limit_mib"`
	SpikeLimitMiB int32  `yaml:"spike_limit_mib"`
}

type Batch struct {
	SendBatchSize int32  `yaml:"send_batch_size"`
	Timeout       string `yaml:"timeout"`
}

type Exporters struct {
	OTLP OTLPExporter `yaml:"otlp"`
}

type OTLPExporter struct {
	Endpoint            string            `yaml:"endpoint"`
	TLS                 TLSConfig         `yaml:"tls"`
	Headers             map[string]string `yaml:"headers,omitempty"`
	QueueSize           int32             `yaml:"queueSize,omitempty"`
	RetryMaxElapsedTime string            `yaml:"retryMaxElapsedTime,omitempty"`
}

type TLSConfig struct {
	Insecure bool `yaml:"insecure"`
}

type Service struct {
	Extensions []string            `yaml:"extensions"`
	Pipelines  map[string]Pipeline `yaml:"pipelines"`
}

type Pipeline struct {
	Receivers  []string `yaml:"receivers"`
	Processors []string `yaml:"processors"`
	Exporters  []string `yaml:"exporters"`
}

// GenerateConfig computes the deterministic ConfigMap content and its hash
func GenerateConfig(profile *telemetryv1alpha1.TelemetryProfile, headerKeys []string) (string, string, error) {
	// Process memory limits
	memLimit := resource.MustParse("128Mi")
	if profile.Spec.Resources.Limits != nil {
		if q, ok := profile.Spec.Resources.Limits[corev1.ResourceMemory]; ok {
			memLimit = q
		}
	}
	limitMiB := int32(memLimit.Value() / (1024 * 1024))
	// Reserve 20% for spike
	spikeLimitMiB := int32(float64(limitMiB) * 0.2)
	limitMiB = limitMiB - 10 // Reserve base overhead

	if limitMiB <= spikeLimitMiB {
		limitMiB = 20
		spikeLimitMiB = 5
	}

	// Process batching
	timeout := "5s"
	if profile.Spec.Batch != nil && profile.Spec.Batch.Timeout != nil {
		timeout = profile.Spec.Batch.Timeout.Duration.String()
	}
	sendBatchSize := int32(512)
	if profile.Spec.Batch != nil && profile.Spec.Batch.SendBatchSize != nil {
		sendBatchSize = *profile.Spec.Batch.SendBatchSize
	}

	queueSize := int32(256)
	if profile.Spec.Exporter.QueueSize != nil {
		queueSize = *profile.Spec.Exporter.QueueSize
	}
	retryMax := "30s"
	if profile.Spec.Exporter.RetryMaxElapsedTime != nil {
		retryMax = profile.Spec.Exporter.RetryMaxElapsedTime.Duration.String()
	}

	cfg := OTelConfig{
		Receivers: Receivers{
			OTLP: OTLPReceiver{
				Protocols: Protocols{
					GRPC: Endpoint{Endpoint: "127.0.0.1:4317"},
					HTTP: Endpoint{Endpoint: "127.0.0.1:4318"},
				},
			},
		},
		Processors: Processors{
			MemoryLimiter: MemoryLimiter{
				CheckInterval: "1s",
				LimitMiB:      limitMiB,
				SpikeLimitMiB: spikeLimitMiB,
			},
			Batch: Batch{
				SendBatchSize: sendBatchSize,
				Timeout:       timeout,
			},
		},
		Exporters: Exporters{
			OTLP: OTLPExporter{
				Endpoint: profile.Spec.Exporter.Endpoint,
				TLS: TLSConfig{
					Insecure: profile.Spec.Exporter.TLS != nil && profile.Spec.Exporter.TLS.Insecure,
				},
				QueueSize:           queueSize,
				RetryMaxElapsedTime: retryMax,
				Headers:             make(map[string]string),
			},
		},
		Service: Service{
			Extensions: []string{"health_check"},
			Pipelines:  make(map[string]Pipeline),
		},
	}

	// Add headers from secret keys
	if profile.Spec.Exporter.HeadersSecretRef != nil && len(headerKeys) > 0 {
		sort.Strings(headerKeys)
		for _, k := range headerKeys {
			cfg.Exporters.OTLP.Headers[k] = fmt.Sprintf("${env:%s}", k)
		}
	}

	// Add pipelines
	signals := profile.Spec.Signals
	if len(signals) == 0 {
		signals = []telemetryv1alpha1.SignalType{
			telemetryv1alpha1.SignalTraces,
			telemetryv1alpha1.SignalMetrics,
			telemetryv1alpha1.SignalLogs,
		}
	}

	// Deterministic ordering of signals
	sigStrs := []string{}
	for _, s := range signals {
		sigStrs = append(sigStrs, string(s))
	}
	sort.Strings(sigStrs)

	for _, s := range sigStrs {
		cfg.Service.Pipelines[s] = Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"memory_limiter", "batch"},
			Exporters:  []string{"otlp"},
		}
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return "", "", err
	}
	configStr := string(b)

	// Compute hash of the config content + profile generation
	h := sha256.New()
	h.Write(b)
	hash := hex.EncodeToString(h.Sum(nil))[:16]

	return configStr, hash, nil
}
