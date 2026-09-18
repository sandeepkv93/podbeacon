# Mission Run Log - M4

## Phase 0-4: Planning
* Investigated `test/e2e/e2e_test.go` and `.github/workflows/test-e2e.yml`.
* Configured `Makefile` to use `kindest/node:v1.31.0` to support modern cert-manager deployments with `selectableFields`.

## Phase 5: Execution
* **T-1**: Wrote `should inject the collector sidecar into an opted-in pod` test block executing `kubectl apply` for `TelemetryProfile` and an opted-in `Pod`, asserting injection behavior natively.
* **T-2**: Evaluated the GitHub Actions `test-e2e.yml` payload, which correctly executes `make test-e2e`.

## Phase 6-7: Verification and Review
* `make test-e2e` uncovered a webhook configuration path mismatch (`/mutate--v1-pod` vs `/mutate-core-v1-pod`). The `kubebuilder` annotation was patched in `pod_webhook.go` and `make manifests` was rerun.
* Principal Review completed: Approved.

## Phase 8: Resolution
* Wait for final verification of `make test-e2e` inside the active `podbeacon-test-e2e` cluster.
