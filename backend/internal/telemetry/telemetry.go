package telemetry

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Init 初始化 W3C 链路传播和可选的 OTLP Trace 导出器。
func Init(ctx context.Context, serviceName, environment, endpoint string) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	options := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(resource.NewSchemaless(
			attribute.String("service.name", serviceName),
			attribute.String("deployment.environment.name", environment),
		)),
	}
	if strings.TrimSpace(endpoint) == "" {
		tracerProvider := sdktrace.NewTracerProvider(options...)
		otel.SetTracerProvider(tracerProvider)
		return tracerProvider.Shutdown, nil
	}
	if err := validateEndpoint(endpoint); err != nil {
		return nil, err
	}

	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}
	options = append(options, sdktrace.WithBatcher(exporter))
	tracerProvider := sdktrace.NewTracerProvider(options...)
	otel.SetTracerProvider(tracerProvider)
	return tracerProvider.Shutdown, nil
}

func validateEndpoint(endpoint string) error {
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT must be a valid HTTP URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT must use http or https")
	}
	return nil
}
