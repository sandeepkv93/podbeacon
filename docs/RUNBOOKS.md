# Runbooks & Operations Guide

## Upgrades

1. Update the image reference in your deployment manifests or Kustomize overlay to the new version.
2. Apply the updated `TelemetryProfile` CRDs:
   ```bash
   kubectl apply -k github.com/sandeepkv93/podbeacon/config/crd
   ```
3. Roll out the updated controller:
   ```bash
   kubectl apply -k github.com/sandeepkv93/podbeacon/config/default
   ```
4. Restart existing injected workloads if the new version requires sidecar updates:
   ```bash
   kubectl rollout restart deployment <my-deployment>
   ```

## Uninstallation

To remove PodBeacon completely from the cluster:
1. Remove the operator and CRDs:
   ```bash
   kubectl delete -k github.com/sandeepkv93/podbeacon/config/default
   ```
2. Any running Pods will retain their sidecars until they terminate. Restart deployments to purge the sidecar:
   ```bash
   kubectl rollout restart deployment <my-deployment>
   ```
3. Manually delete lingering profiles if the finalizers were stuck (optional).

## Troubleshooting

### Symptom: Pod creation hangs or fails with "Internal Error" from webhook
**Cause:** The mutating webhook is unreachable, or `cert-manager` failed to inject the CA bundle.
**Resolution:**
1. Check `cert-manager` logs: `kubectl logs -n cert-manager -l app=cert-manager`.
2. Check if the PodBeacon controller is running: `kubectl get pods -n podbeacon-system`.
3. Check webhook certificate: `kubectl get secret webhook-server-cert -n podbeacon-system`.

### Symptom: Pods schedule but without the sidecar
**Cause:** Pod does not have the exact opt-in annotation, or is in the `kube-system` namespace.
**Resolution:**
Ensure the Pod spec contains:
```yaml
annotations:
  telemetry: "enable"
```

### Symptom: Webhook fails with "TelemetryProfile default not found"
**Cause:** You annotated a Pod but did not create a `TelemetryProfile` in that namespace.
**Resolution:**
Create a `TelemetryProfile` named `default` in the same namespace as your Pod, or specify the profile explicitly via the `podbeacon.io/profile: my-profile` annotation.
