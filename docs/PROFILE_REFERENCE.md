# TelemetryProfile Reference

The `TelemetryProfile` Custom Resource Definition (CRD) configures the OpenTelemetry Collector sidecars injected by PodBeacon.

## API Group and Version
`telemetry.podbeacon.io/v1alpha1`

## Resource Scope
`Namespaced` - Profiles only apply to Pods within the same namespace.

## Schema

### `spec.exporter` (Required)
Configures the OTLP destination for the collector.

- `endpoint` (String, Required): The OTLP/gRPC host and port (e.g., `my-otel-collector:4317`).
- `tls` (Object, Optional): TLS configuration.
  - `insecure` (Boolean): Disable TLS verification (default: false).
  - `caSecretRef` (Object): Reference to a Secret in the same namespace containing a `ca.crt` key.
- `headersSecretRef` (Object, Optional): Reference to a Secret whose keys map to outbound OTLP gRPC headers.
- `queueSize` (Integer, Optional): Number of batches to queue (default: 256).
- `retryMaxElapsedTime` (Duration, Optional): Max time spent retrying a batch (default: "30s").

### `spec.signals` (Optional)
List of signals to collect and export. Available options are `traces`, `metrics`, and `logs`. By default, all three are collected.

### `spec.resources` (Optional)
Standard Kubernetes `ResourceRequirements` (requests/limits for cpu/memory) for the injected sidecar. Defaults to 50m CPU, 64Mi Memory requests; 200m CPU, 256Mi Memory limits.

### `spec.batch` (Optional)
Configures the batch processor.
- `timeout` (Duration, Optional): Time to wait before flushing (default: "5s").
- `sendBatchSize` (Integer, Optional): Number of spans/metrics/logs to batch (default: 512).

## Status

The `status` field provides observability into the profile's state:
- `conditions`: Array of status conditions (e.g., `Ready=True`).
- `configMapName`: Name of the generated ConfigMap containing the collector configuration that the sidecars will mount.
