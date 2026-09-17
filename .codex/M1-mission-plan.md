# Mission Plan - M1: Profile Controller

## Objective
Implement Milestone M1 (Profile controller) for PodBeacon.
Success criteria: Generate the `TelemetryProfile` CRD, implement typed validation/defaulting, implement status, implement deterministic config revisions (ConfigMap hashing and rendering), and perform Secret checks. The output must pass API/CFG integration tests (`envtest`) without needing a webhook.

## Trigger context
- Trigger: User requested to move to the next steps using the `$mission-ops` skill.
- Source: Chat.
- Requirements: Complete M1 as defined in `docs/REQUIREMENTS.md`.
- Classification: Feature (High confidence).

## Recon Summary
- Repository: `/home/sandeepkv/dev/go-projects/podbeacon`
- Branch: main (unborn)
- State: `kubebuilder init` completed.
- Next action: run `kubebuilder create api`.

## Task DAG
| Task ID | Goal | Owner | Dependencies | File Ownership | Risk Tier |
|---|---|---|---|---|---|
| T-1 | Scaffold CRD and Controller | executor | none | api/, internal/controller/ | 0 |
| T-2 | Implement `TelemetryProfile` typed schema and validation | executor | T-1 | api/v1alpha1/telemetryprofile_types.go, api/v1alpha1/telemetryprofile_webhook.go (if needed for defaulting) | 1 |
| T-3 | Implement Config rendering (relay.yaml generation and hashing) | executor | T-2 | internal/controller/config_render.go | 1 |
| T-4 | Implement Reconciliation logic (ConfigMap materialization, Secret checks, Status updates) | executor | T-3 | internal/controller/telemetryprofile_controller.go | 1 |
| T-5 | Write envtest Integration Tests | executor | T-4 | internal/controller/telemetryprofile_controller_test.go | 1 |
| T-6 | Principal Review | principal_reviewer | T-5 | .codex/principal-review.md, .codex/principal-review.json | 1 |
| T-7 | Review Resolution and Closeout | coordinator | T-6 | .codex/ | 0 |

## Risk and Rollback
| Task ID | Primary Risk | Detection Signal | Mitigation | Rollback |
|---|---|---|---|---|
| T-2 | Schema invalid or defaulting fails | `make manifests` or `make test` fails | Unit test defaulting/validation | `git checkout` |
| T-3 | Config rendering generates invalid YAML | Integration tests fail to parse config | Use typed structs for YAML rendering | `git checkout` |
| T-4 | Controller gets stuck in a retry loop | `make test` times out | Set appropriate backoff and state | `git checkout` |
| T-5 | Tests are flaky | Repeated `make test` fails randomly | Ensure determinism | Fix tests |

## Validation Plan
| Task ID | Required Checks | Evidence Required |
|---|---|---|
| T-1 | `kubebuilder create api` success | Run log |
| T-2 | `make manifests` runs cleanly | Run log |
| T-3 | Unit tests for config rendering | `make test` output |
| T-4, T-5 | envtest suites pass | `make test` output |
| T-6 | Principal review | Review JSON |

## Gates
Preflight: PASS.
Execution mode: single-agent.
Batch/final: `make test` and `make manifests` are clean.
Principal review: approved decision required.
