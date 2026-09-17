# PodBeacon

A Kubernetes operator in Go that automatically provisions and configures
OpenTelemetry collector sidecars for pods with the annotation:

```yaml
metadata:
  annotations:
    telemetry: "enable"
```

## Status

Initial Go project only. Operator behavior and Kubernetes integration are not
implemented yet.

## Development

Requires Go 1.25.4 or later.

```sh
go build ./...
go test ./...
go vet ./...
```
