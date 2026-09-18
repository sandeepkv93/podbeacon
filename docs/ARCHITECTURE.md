# Architecture

PodBeacon implements a Kubernetes Operator pattern specifically tailored to inject and configure OpenTelemetry Collector sidecars natively.

## Core Components

1. **Profile Controller (`TelemetryProfile` Reconciler)**
   - Watches `TelemetryProfile` Custom Resources.
   - Idempotently renders a standard OpenTelemetry Collector configuration YAML based on the profile spec.
   - Materializes this configuration into an immutable `ConfigMap`.
   - Updates the `TelemetryProfile` status with the target `ConfigMap` name and the `Ready` condition.

2. **Pod Mutating Webhook (`PodDefaulter`)**
   - Intercepts Pod `CREATE` requests.
   - Checks for the exact opt-in annotation: `telemetry: "enable"`.
   - Locates the corresponding `TelemetryProfile` in the Pod's namespace (defaulting to the profile named `default`).
   - Rejects the Pod creation if the profile does not exist or is not `Ready` (failurePolicy: Fail).
   - Patches the Pod spec to append a Kubernetes 1.28+ native sidecar (`initContainer` with `restartPolicy: Always`).
   - Mounts the profile's generated `ConfigMap` into the sidecar.

## Design Decisions
- **Native Sidecars:** We utilize Kubernetes native sidecars (via `initContainers`) to ensure the collector starts before the application and survives Job completions gracefully.
- **Immutable ConfigMaps:** By generating new ConfigMaps for profile updates, existing Pods are protected from in-place disruptions. Updates are rolled out predictably via normal deployment rolling updates.
- **Strict Opt-In:** Sidecars are never automatically applied to a whole namespace without the explicit Pod-level `telemetry: "enable"` annotation.
