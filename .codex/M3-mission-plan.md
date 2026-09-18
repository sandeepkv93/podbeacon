# Mission Plan - M3: Webhook RBAC and Certification

## Objective
Implement Milestone M3: Ensure operator scoped RBAC includes `get profiles, secrets` and integrate cert-manager annotations on the `MutatingWebhookConfiguration`.

## Trigger context
- Trigger: User requested to proceed with M3 using `$mission-ops`
- Source: Chat
- Classification: Implementation configuration.

## Recon Summary
- The `+kubebuilder:rbac` markers for secrets and profiles are already defined in `telemetryprofile_controller.go` and `pod_webhook.go`.
- Cert-manager patches are already active in `config/default/kustomization.yaml` because of the webhook scaffolding from M2.

## Task DAG
| Task ID | Goal | Owner | Dependencies | File Ownership | Risk Tier |
|---|---|---|---|---|---|
| T-1 | Validate RBAC roles | executor | none | config/rbac/ | 0 |
| T-2 | Validate Cert-Manager Kustomize pipeline | executor | none | config/default/kustomization.yaml | 0 |
| T-3 | Principal Review | principal_reviewer | T-1, T-2 | .codex/ | 1 |
| T-4 | Closeout | coordinator | T-3 | .codex/ | 0 |

## Validation Plan
Run `kubectl kustomize config/default` to ensure the final aggregated deployment YAML has `cert-manager.io/inject-ca-from` and RBAC roles with `secrets` permissions.
