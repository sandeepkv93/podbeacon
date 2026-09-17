# Certificate Packaging Spike

## Premise
The PodBeacon mutating webhook requires a TLS certificate to serve admission requests. The Kubernetes API server must trust the Certificate Authority (CA) that signed the webhook's certificate.

## Solution: cert-manager
We use `cert-manager` to generate a self-signed CA and a serving certificate for the webhook. `cert-manager` automatically injects the CA bundle into the `MutatingWebhookConfiguration` via an annotation.

## Spike Manifests

### 1. Certificate Resources
```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: podbeacon-selfsigned-issuer
  namespace: podbeacon-system
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: podbeacon-serving-cert
  namespace: podbeacon-system
spec:
  dnsNames:
  - podbeacon-webhook-service.podbeacon-system.svc
  - podbeacon-webhook-service.podbeacon-system.svc.cluster.local
  issuerRef:
    kind: Issuer
    name: podbeacon-selfsigned-issuer
  secretName: webhook-server-cert
```

### 2. MutatingWebhookConfiguration
```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: podbeacon-mutating-webhook-configuration
  annotations:
    cert-manager.io/inject-ca-from: podbeacon-system/podbeacon-serving-cert
webhooks:
- name: podbeacon.podbeacon.io
  clientConfig:
    service:
      name: podbeacon-webhook-service
      namespace: podbeacon-system
      path: /mutate-v1-pod
  rules:
  - operations: ["CREATE"]
    apiGroups: [""]
    apiVersions: ["v1"]
    resources: ["pods"]
  sideEffects: None
  admissionReviewVersions: ["v1"]
  failurePolicy: Fail
  timeoutSeconds: 3
```

## Validation
1. `cert-manager` must be installed prior to PodBeacon installation.
2. The `Certificate` resource creates a Kubernetes Secret named `webhook-server-cert` containing `tls.crt` and `tls.key`.
3. The operator mounts this Secret into the webhook server pod at `/tmp/k8s-webhook-server/serving-certs`.
4. The API Server reads the CA bundle from the `MutatingWebhookConfiguration` to verify the webhook connection.
