# Mission Plan

## Objective
Execute Milestone M0 (Prove platform assumptions). Create runnable spike evidence (YAMLs) to validate restartable sidecar probes, collector memory settings, Secret-backed headers, and certificate packaging assumptions. Write a design document resolving the Section 14 blockers from docs/REQUIREMENTS.md.
Success: Spike YAMLs in `docs/spike/` that demonstrate the lifecycle and configuration constraints, plus a `docs/M0-decisions.md` document addressing all Section 14 blockers.

## Trigger context
- Trigger: user asks to understand requirements and use /mission-ops to do what needs to be done.
- Source: current conversation.
- Requirements: Complete the next logical milestone (M0) before writing Go code.
- Classification: refactor/feature (Spike and design document), High confidence.

## Recon Summary
- Repository: /home/sandeepkv/dev/go-projects/podbeacon
- Branch: main
- Go module: github.com/sandeepkv93/podbeacon; Go 1.25.4
- Only documentation exists. `docs/REQUIREMENTS.md` outlines M0-M4. We are at M0.
- `kind` is available (version 0.20.0). `kubebuilder` is not installed yet (we don't need it for the M0 spike YAMLs, we just need to specify the version to pin).

## Task DAG
| Task ID | Goal | Owner | Dependencies | File Ownership | Risk Tier |
|---|---|---|---|---|---|
| T-1 | Spike: Native sidecar lifecycle & resource limits | executor | none | docs/spike/sidecar-lifecycle.yaml | 0 |
| T-2 | Spike: Secret-to-header mechanism | executor | none | docs/spike/secret-headers.yaml | 0 |
| T-3 | Spike: Certificate packaging & cert-manager | executor | none | docs/spike/cert-manager-packaging.md | 0 |
| T-4 | Resolve Section 14 blockers | executor | T-1, T-2, T-3 | docs/M0-decisions.md | 0 |
| T-5 | Independent principal review | principal_reviewer | T-4 | .codex/principal-review.md, .codex/principal-review.json | 0 |
| T-6 | Resolve findings and closeout | coordinator | T-5 | .codex/ | 0 |

## Risk and Rollback
| Task ID | Primary Risk | Detection Signal | Mitigation | Rollback |
|---|---|---|---|---|
| T-1 | YAML syntax errors | YAML validation fails | Lint YAML | Delete file |
| T-2 | OTel configuration invalid | OTel collector fails to start | Validate OTel config schema visually | Delete file |
| T-3 | Incorrect cert-manager assumptions | Logic gaps in docs | Check cert-manager docs | Delete file |
| T-4 | Missed section 14 items | Reviewer catches missing items | Checklist vs REQUIREMENTS.md | Revert edits |

## Validation Plan
| Task ID | Required Checks | Evidence Required |
|---|---|---|
| T-1, T-2, T-3 | YAML validation (e.g., yq or dry-run) | Command output in run-log |
| T-4 | Manual checklist against Section 14 | Included in run-log |
| T-5 | Principal review (Architecture, correctness) | JSON/MD artifacts |
| T-6 | Zero open findings | run-log |

## Gates
Preflight: PASS.
Execution mode: single-agent. (Scope is small documentation and YAMLs, easy to execute sequentially).
Batch/final: Spike YAMLs and decisions document are complete.
Principal review: approved decision required. Review resolution: zero open findings.
