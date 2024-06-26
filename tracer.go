package gcptrace

import (
	"context"

	"cloud.google.com/go/compute/metadata"
	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/api/option"
)

func InitTracing(ctx context.Context, tpOptions ...trace.TracerProviderOption) (func(), error) {
	res, err := resource.New(ctx,
		resource.WithDetectors(gcp.NewDetector()),
		resource.WithTelemetrySDK(),
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, err
	}

	var exporter trace.SpanExporter
	if !metadata.OnGCE() {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
	} else {
		exporter, err = cloudtrace.New(
			// we disable telemetry to avoid infinite loop when using OpenCensus bridge
			// https://github.com/open-telemetry/opentelemetry-go/issues/1928
			cloudtrace.WithTraceClientOptions([]option.ClientOption{option.WithTelemetryDisabled()}),
		)
	}
	if err != nil {
		return nil, err
	}

	opts := []trace.TracerProviderOption{
		trace.WithResource(res),
		// Span processor here extracts SampleRate from baggage and adds it as attribute to all spans.
		trace.WithSpanProcessor(SampleRateAnnotator{}),
		trace.WithBatcher(exporter),
	}

	opts = append(opts, tpOptions...)

	// Create a new tracer provider with resource and batched otlp exporter
	tp := trace.NewTracerProvider(opts...)

	otel.SetTracerProvider(tp)

	// Per otel specification default propagator is no-op.
	// We use the contrib autoprop package to install some default propagators.
	// List of propagators can be overridden by setting OTEL_PROPAGATORS environment variable.
	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())

	return func() {
		_ = tp.Shutdown(ctx)
	}, nil
}

// SampleRateAnnotator is a SpanProcessor that adds baggage SampleRate as attribute to all started spans.
type SampleRateAnnotator struct{}

func (a SampleRateAnnotator) OnStart(ctx context.Context, s trace.ReadWriteSpan) {
	b := baggage.FromContext(ctx)
	sampleRateStr := b.Member("SampleRate").Value()
	if sampleRateStr == "" {
		return
	}
	s.SetAttributes(attribute.String("SampleRate", sampleRateStr))
}
func (a SampleRateAnnotator) Shutdown(context.Context) error   { return nil }
func (a SampleRateAnnotator) ForceFlush(context.Context) error { return nil }
func (a SampleRateAnnotator) OnEnd(s trace.ReadOnlySpan)       {}
