# Mission Plan - M4: E2E Delivery Pipeline

## Objective
Implement M4: Create an end-to-end Ginkgo test suite that deploys the operator and tests Pod injection, integrated with GitHub Actions.

## Trigger context
- Trigger: User prompt
- Scope: `test/e2e/e2e_test.go` and `.github/workflows/test-e2e.yml`
- Execution mode: single-agent

## Recon Summary
- The `test-e2e.yml` action natively executes `make test-e2e`.
- `Makefile` scaffolds the Kind cluster and runs the `e2e` suite.
- Replaced the node image with `kindest/node:v1.31.0` in `Makefile` to accommodate newer cert-manager dependencies.
- Inserted injection integration verification into `test/e2e/e2e_test.go`.

## Task DAG
| Task ID | Goal | Owner | Dependencies | File Ownership | Risk Tier |
|---|---|---|---|---|---|
| T-1 | Inject E2E verification test | executor | none | test/e2e/e2e_test.go | 1 |
| T-2 | Ensure GitHub action setup | executor | none | .github/workflows/test-e2e.yml | 0 |
| T-3 | Principal Review | principal_reviewer | T-1, T-2 | .codex/ | 1 |
| T-4 | Closeout | coordinator | T-3 | .codex/ | 0 |

## Validation Plan
Ensure `make test-e2e` passes locally on the Kind cluster and verify the test applies the `TelemetryProfile`, spawns a test Pod with `telemetry: enable`, and checks for `podbeacon-collector` injection.
