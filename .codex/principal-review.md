# Principal Review - M2

## Review Context
* **Mission Objective**: Execute Milestone M2 (Injection Webhook)
* **Changes**: Generated `MutatingWebhookConfiguration` and implemented the Pod mutating logic via a generic Defaulter.

## Review Perspectives

### 1. Architecture
* The mutation targets `CREATE` operations on `core/v1/Pod` via the `MutatingWebhookConfiguration`.
* Uses `InitContainers` with `RestartPolicy: Always` (the native sidecar approach).
* Correctly depends on `TelemetryProfile` availability in the same namespace, avoiding implicit cross-namespace boundaries.

### 2. Correctness
* Exact opt-in `telemetry: "enable"` is checked case-sensitively.
* HostNetwork pods are correctly rejected per INJ-08.
* Container conflicts (e.g., if a pod already has a container named `podbeacon-collector`) correctly result in a denied admission.
* Resource requests and limits default correctly if omitted from the profile.
* Idempotency is verified using the `podbeacon.io/injected: "true"` annotation.

### 3. Performance / Security
* The webhook fails open (`failurePolicy: fail` by default as specified in ADM-02) ensuring unannotated pods are not silently accepted if the operator is down, but this requires namespace scope filtering for operators.
* `Readonly` is explicitly `true` for the ConfigMap mount.
* It leverages `client.Client` only to get the cached profile without calling out to external services, ensuring low latency.

### 4. Operability
* Pods are annotated with the profile UID and config hash (`podbeacon.io/profile-uid`, `podbeacon.io/config-hash`) providing provenance for the injected sidecar.
* The tests verify successful injection on opted-in pods and successful bypass on unannotated pods.

## Decision
**Status**: APPROVED

The injection webhook fulfills all M2 goals.
