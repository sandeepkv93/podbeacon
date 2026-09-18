# Security and Reliability Considerations

PodBeacon is designed with strict boundaries to limit blast radius and adhere to restricted security standards.

## Restricted Pod Security Standards
The injected sidecar is compliant with the Kubernetes `Restricted` Pod Security Standard:
- `runAsNonRoot: true`
- `allowPrivilegeEscalation: false`
- Drops all Linux capabilities (`ALL`).
- Sets `seccompProfile: RuntimeDefault`.
- Optionally mounts a read-only root filesystem.

## Namespaced Profiles
A `TelemetryProfile` is strictly scoped to its namespace. A developer in `namespace-a` cannot mount a profile or reference a Secret located in `namespace-b`.

## Idempotency and Webhook Reliability
- The `MutatingWebhookConfiguration` matches only Pod `CREATE` events. It does not intercept `UPDATE` requests, avoiding infinite loops or disruption of running Pods.
- The webhook explicitly ignores Pods with `hostNetwork: true` to prevent unintended port bindings.
- It uses a `failurePolicy: Fail` to ensure that an opted-in Pod is *never* deployed without observability running. If the control plane is down, Pods requesting telemetry will not schedule until observability can be guaranteed.
- It checks for the `podbeacon.io/injected: "true"` annotation to prevent duplicate sidecar injections during reinvocations.

## RBAC Permissions
The PodBeacon controller requires minimal RBAC permissions:
- Read/Watch `Pods`.
- Read/Watch/Update `TelemetryProfile`.
- Create/Update/Delete/List `ConfigMaps` owned by the operator.
- Read `Secrets` (only to verify presence of user-specified CA or Header secrets).
