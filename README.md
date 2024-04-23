# GCP Trace package

[![PkgGoDev][godev:image]][godev:url]

This package contains shared initialization code that exports collected spans via otlp exporter.

```bash
go get "github.com/MottoStreaming/gcptrace.go"
```

## Initialization example:

Deployment example:
```yaml
env:
  - name: POD_NAME
    valueFrom:
      fieldRef:
        fieldPath: metadata.name
  - name: NAMESPACE_NAME
    valueFrom:
      fieldRef:
        fieldPath: metadata.namespace
  - name: CONTAINER_NAME
    value: my-container-name
  - name: OTEL_RESOURCE_ATTRIBUTES
    value: k8s.pod.name=$(POD_NAME),k8s.namespace.name=$(NAMESPACE_NAME),k8s.container.name=$(CONTAINER_NAME),SampleRate=10
  - name: OTEL_SERVICE_NAME
    value: my-service-name
  - name: OTEL_SERVICE_VERSION
    value: xxyyzz
  - name: OTEL_TRACES_SAMPLER
    value: parentbased_traceidratio
  - name: OTEL_TRACES_SAMPLER_ARG
    value: 0.1 # value must match with SampleRate attribute
```

main.go:

```go
func main() {
    ctx := context.Background()
    shutdown, err := gcptrace.InitTracing(ctx)
    if err != nil {
        log.Fatalf("unable to set up tracing: %v", err)
    }
    defer shutdown()
}
```

## Tracing your own code (storage layer in this example):

```go
import (
    "go.opentelemetry.io/otel"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
    "go.opentelemetry.io/otel/trace"
)

// Inside constructor initialize separate instance of tracer 
tracer:  otel.Tracer("")

func (s *storage) startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
    spanCtx, span := s.tracer.Start(ctx,
        name,
        trace.WithSpanKind(trace.SpanKindInternal),
        trace.WithAttributes(
            semconv.DBSystemPostgreSQL,
        ),
    )
    return spanCtx, span
}

func (s *storage) GetDRMTechnologies(ctx context.Context, streamID string) ([]*bff_v1.Event, error) {
    spanCtx, span := s.startSpan(ctx, "GetDRMTechnologies")
    defer span.End()
    span.SetAttributes(attribute.String("stream_id", streamID))
    ...
    if err != nil {
        // Record error in the span
        span.RecordError(err)
        return nil, errors.Wrap(err, "failed to read playlist DRM info")
    }
}
```

[godev:image]: https://pkg.go.dev/badge/github.com/MottoStreaming/gcptrace.go
[godev:url]:   https://pkg.go.dev/github.com/MottoStreaming/gcptrace.go
