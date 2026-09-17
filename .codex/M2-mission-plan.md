# Mission Plan - M2: Injection Webhook

## Objective
Implement Milestone M2 (Injection webhook) for PodBeacon.
Success criteria: Generate the MutatingWebhookConfiguration, implement Pod sidecar injection logic (exact opt-in `telemetry: "enable"`, namespace exclusions, native sidecar patch, preserved fields, conflict handling, and idempotency), and ensure INJ/ADM unit/integration tests pass.

## Trigger context
- Trigger: User requested to work on M2 using the `$mission-ops` skill.
- Source: Chat.
- Requirements: Complete M2 as defined in `docs/REQUIREMENTS.md`.
- Classification: Feature (High confidence).

## Recon Summary
- Repository: `/home/sandeepkv/dev/go-projects/podbeacon`
- State: M1 completed. `TelemetryProfile` CRD and controller implemented.
- Next action: run `kubebuilder create webhook` for core/v1 Pod.

## Task DAG
| Task ID | Goal | Owner | Dependencies | File Ownership | Risk Tier |
|---|---|---|---|---|---|
| T-1 | Scaffold Pod Webhook | executor | none | internal/webhook/, config/webhook/ | 0 |
| T-2 | Implement exact opt-in and conflict checks | executor | T-1 | internal/webhook/v1/pod_webhook.go | 1 |
| T-3 | Implement sidecar injection mutation | executor | T-2 | internal/webhook/v1/pod_webhook.go | 1 |
| T-4 | Write webhook unit and integration tests | executor | T-3 | internal/webhook/v1/pod_webhook_test.go | 1 |
| T-5 | Principal Review | principal_reviewer | T-4 | .codex/principal-review.md, .codex/principal-review.json | 1 |
| T-6 | Review Resolution and Closeout | coordinator | T-5 | .codex/ | 0 |

## Risk and Rollback
| Task ID | Primary Risk | Detection Signal | Mitigation | Rollback |
|---|---|---|---|---|
| T-1 | Scaffolding error with external type | Command fails | Follow Kubebuilder docs carefully | `git checkout` |
| T-2 | Incorrect opt-in logic | Unit test fails | TDD for exact matches | Fix logic |
| T-3 | Incorrect Pod patch generation | Webhook errors | Validate JSON patch | Fix logic |
| T-4 | Envtest configuration fails for webhook | `make test` fails | Ensure certs are generated in envtest | Fix tests |

## Validation Plan
| Task ID | Required Checks | Evidence Required |
|---|---|---|
| T-1 | `kubebuilder create webhook` success | Run log |
| T-2, T-3, T-4 | `make test` runs cleanly in webhook dir | `make test` output |
| T-5 | Principal review | Review JSON |

## Gates
Preflight: PASS.
Execution mode: single-agent.
Batch/final: `make test` and `make manifests` are clean.
Principal review: approved decision required.
