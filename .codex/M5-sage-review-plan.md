# Mission Plan - M5: Sage Review Remediation

## Objective
Address all 15 Critical, Major, and Minor findings detailed in `/tmp/podbeacon-sage-review.md` to ensure the PodBeacon operator is robust, secure, and ready for release.

## Trigger context
- Trigger: User provided a comprehensive Sage Review with 15 findings.
- Scope: Controller, Webhook, CRD, E2E Tests, Makefile.
- Execution mode: multi-agent team

## Recon Summary
- The review identified 4 Critical, 10 Major, and 1 Minor findings.
- Key themes: Webhook blast radius (missing selectors), provenance/validation bypasses, missing CA injection, incomplete E2E isolation, and missing CRD structural validations.
- Actionable steps map cleanly to Webhook logic, Controller logic, CRD API, and E2E/Makefile updates.

## Task DAG
| Task ID | Goal | Owner | Dependencies | File Ownership | Risk Tier |
|---|---|---|---|---|---|
| T-1 | Webhook Blast Radius & Provenance (Findings 1, 2, 3, 11) | webhook_engineer | none | internal/webhook/v1/ | 3 |
| T-2 | Controller Revisions & Configuration (Findings 4, 5, 7, 8, 10) | controller_engineer | none | internal/controller/ | 3 |
| T-3 | Controller Watches & Readiness (Findings 6, 9) | controller_engineer | T-2 | internal/controller/, cmd/main.go | 2 |
| T-4 | CRD Structural Validation (Finding 12) | api_engineer | none | api/v1alpha1/ | 2 |
| T-5 | E2E Isolation & Artifact Cleanup (Findings 13, 15) | test_engineer | none | Makefile, test/utils/, .gitignore | 1 |
| T-6 | Expanded E2E Matrix (Finding 14) | test_engineer | T-1, T-2, T-3, T-4, T-5 | test/e2e/ | 2 |
| T-7 | Principal Review | principal_reviewer | T-6 | .codex/ | 1 |
| T-8 | Closeout | coordinator | T-7 | .codex/ | 0 |

## Validation Plan
Ensure `make test-e2e` passes with expanded isolation and comprehensive matrices. Ensure envtest unit tests cover all webhook and controller edge cases.
