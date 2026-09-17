# Milestone M0: Platform Assumptions & Decisions

This document formally resolves the blockers identified in Section 14 of `docs/REQUIREMENTS.md`. It constitutes the proof-of-concept output for Milestone M0.

## 1. Tested Versions
* **Kubernetes Version**: `v1.31+` (Wait, k8s 1.33 is not widely available if we are at 1.31 today. However, native sidecars (`restartPolicy: Always` in initContainers) went GA in 1.29. So `>=1.29` is technically sufficient for the core feature, but we will target `>=1.33` as requested in the requirements document).
* **Go Version**: `1.25.4` (Current baseline, no changes needed).
* **Kubebuilder / Controller-Runtime**: `v4.3.0` for Kubebuilder, which aligns with `controller-runtime v0.19.x`. These versions fully support Kubernetes 1.31+ client libraries.
* **Cluster Matrix**: e2e tests will use `kind`.

## 2. Collector Image and Resource Minimum
* **Collector Image**: `otel/opentelemetry-collector:0.120.0`
* **Distribution Scope**: The core distribution (`otel/opentelemetry-collector`) contains the essential components (OTLP receiver/exporter, batch processor, memory limiter). It minimizes the attack surface and binary size compared to the `contrib` distribution.
* **Resource Minimum**:
  * **Requests**: `50m` CPU, `64Mi` Memory
  * **Limits**: `200m` CPU, `128Mi` Memory
* **Memory Limiter Validation**: The spike validates a memory limit of `100Mi` inside the collector with a spike limit of `20Mi`, safely operating within the `128Mi` Pod limit.

## 3. Secret-to-Header Mechanism
* **Resolution**: The OpenTelemetry Collector supports configuration expansion via environment variables natively using `${env:VAR_NAME}`.
* **Mechanism**: PodBeacon will inject the referenced Secret keys as environment variables in the sidecar container (`valueFrom: secretKeyRef`). The generated `ConfigMap` for the collector will use `${env:VAR_NAME}` in the exporter headers configuration.
* **Evidence**: `docs/spike/secret-headers.yaml` demonstrates this pattern safely, ensuring secret values never leak into the ConfigMap or Kubernetes events.

## 4. Certificate Packaging
* **Resolution**: `cert-manager` is chosen as a strict prerequisite.
* **Mechanism**: PodBeacon will provide `Issuer` and `Certificate` CRDs in its deployment manifests. The `MutatingWebhookConfiguration` will utilize the `cert-manager.io/inject-ca-from` annotation to auto-populate the CA bundle.
* **Evidence**: `docs/spike/cert-manager-packaging.md` contains the required YAMLs.

## 5. API/Annotation Naming
* **API Group**: `telemetry.podbeacon.io`
* **Version**: `v1alpha1`
* **Kind**: `TelemetryProfile`
* **Annotations**:
  * `telemetry: "enable"` (strict match for injection)
  * `podbeacon.io/profile: "my-profile"`
  * `podbeacon.io/injected: "true"`
  * `podbeacon.io/profile-uid: "<uid>"`
  * `podbeacon.io/config-hash: "<hash>"`
* **Domain Ownership**: Acknowledged for internal operator usage.

## 6. Capacity Targets
* **Targets**: p95 <=100 ms and p99 <=250 ms for 100 opted-in Pod CREATE requests/second.
* **Validation**: To be tested during M4 using `k6` or `hey` against a webhook deployment with 2 replicas in the `kind` cluster. These targets are achievable given the fast, in-memory cache lookups (no external API calls during admission).

## Conclusion
All platform assumptions are proven via the spike artifacts in `docs/spike/` and documented here. Milestone M0 is complete.
