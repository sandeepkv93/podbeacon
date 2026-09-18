# Run Log - M5 Sage Review Remediation

- **T-1 (Webhook Engineer)**: Added `namespaceSelector`, `objectSelector`, and `timeoutSeconds`. Replaced blind annotation check with shape verification. Configured Downward API and Liveness Probe for sidecar.
- **T-2 (Controller Engineer)**: Mapped Secret header keys to valid environment variables. Enforced `immutable: true` on ConfigMaps. Added comprehensive configHash including credentials and image digests.
- **T-3 (Controller Engineer)**: Set controller replicas to 2. Configured dynamic readiness check using `mgr.GetWebhookServer().StartedChecker()`. Added `handler.EnqueueRequestsFromMapFunc` watches for dependent Secrets and ConfigMaps.
- **T-4 (API & Test Engineer)**: Added CEL structural validations for resource limits, queue size boundaries, and explicit list lengths.
- **T-5 (API & Test Engineer)**: Removed tracked `kubebuilder` binary. Ignored `bin/` and `cover.out`. Rebuilt `Makefile` isolated run context using `E2E_KUBECONFIG`.
- **T-6 (API & Test Engineer)**: Expanded E2E suite to execute validation over DaemonSets, CronJobs, and StatefulSets.
- **T-7**: Principal Review conducted and approved.
- **T-8**: Closeout artifacts generated.
