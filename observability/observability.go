// Package observability provides OpenTelemetry integration for GoKit packages.
//
// This package offers tracing, metrics, and structured logging capabilities
// that can be used across all GoKit modules for comprehensive observability.
//
// Features:
//   - Distributed tracing with OpenTelemetry
//   - Custom metrics for performance monitoring
//   - Structured logging with correlation IDs
//   - Context-aware observability
//   - Zero-dependency when not configured
//
// Example usage:
//
//	import "github.com/kdsmith18542/gokit/observability"
//
//	func main() {
//	    // Initialize observability (optional)
//	    observability.Init(observability.Config{
//	        ServiceName: "my-app",
//	        ServiceVersion: "1.0.0",
//	        Environment: "production",
//	    })
//
//	    // Use in your application
//	    ctx := context.Background()
//	    span := observability.StartSpan(ctx, "user_registration")
//	    defer span.End()
//
//	    // Add metrics
//	    observability.RecordMetric("form_validations", 1, map[string]string{
//	        "form_type": "registration",
//	        "status": "success",
//	    })
//	}
package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds the configuration for observability initialization
type Config struct {
	// ServiceName is the name of the service for tracing and metrics
	ServiceName string
	// ServiceVersion is the version of the service
	ServiceVersion string
	// Environment is the deployment environment (dev, staging, prod)
	Environment string
	// EnableTracing enables distributed tracing
	EnableTracing bool
	// EnableMetrics enables metrics collection
	EnableMetrics bool
	// EnableLogging enables structured logging
	EnableLogging bool
}

// Observer provides observability capabilities for GoKit operations
type Observer interface {
	// Form validation observability
	OnFormValidationStart(ctx context.Context, formName string)
	OnFormValidationEnd(ctx context.Context, formName string, errorCount int, duration time.Duration)
	OnFormValidationError(ctx context.Context, formName string, field string, error string)

	// i18n observability
	OnTranslationStart(ctx context.Context, locale string, key string)
	OnTranslationEnd(ctx context.Context, locale string, key string, duration time.Duration)
	OnLocaleDetection(ctx context.Context, detectedLocale string, fallbackUsed bool)

	// Upload observability
	OnUploadStart(ctx context.Context, fileName string, fileSize int64)
	OnUploadEnd(ctx context.Context, fileName string, fileSize int64, duration time.Duration, success bool)
	OnUploadError(ctx context.Context, fileName string, error string)
	OnStorageOperation(ctx context.Context, operation string, storageType string, duration time.Duration, success bool)
}

// Global observer instance
var globalObserver Observer = &noopObserver{}

// Init initializes the observability system with the given configuration
func Init(config Config) error {
	if !config.EnableTracing && !config.EnableMetrics && !config.EnableLogging {
		// No observability enabled, use no-op observer
		return nil
	}

	// Initialize OpenTelemetry if tracing or metrics are enabled
	if config.EnableTracing || config.EnableMetrics {
		if err := initOpenTelemetry(config); err != nil {
			return fmt.Errorf("failed to initialize OpenTelemetry: %w", err)
		}
	}

	// Create and set the observer
	observer := &otelObserver{
		config: config,
		tracer: otel.Tracer("gokit"),
		meter:  otel.Meter("gokit"),
	}
	globalObserver = observer

	return nil
}

// SetObserver sets a custom observer for observability events
func SetObserver(observer Observer) {
	globalObserver = observer
}

// GetObserver returns the current observer instance
func GetObserver() Observer {
	return globalObserver
}

// StartSpan starts a new span for tracing
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer("gokit").Start(ctx, name, opts...)
}

// RecordMetric records a metric with the given name, value, and attributes
func RecordMetric(name string, value float64, attributes map[string]string) {
	if observer, ok := globalObserver.(*otelObserver); ok {
		observer.recordMetric(name, value, attributes)
	}
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(ctx context.Context, name string, attributes map[string]string) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		attrs := make([]attribute.KeyValue, 0, len(attributes))
		for k, v := range attributes {
			attrs = append(attrs, attribute.String(k, v))
		}
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SetSpanAttributes sets attributes on the current span
func SetSpanAttributes(ctx context.Context, attributes map[string]string) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		attrs := make([]attribute.KeyValue, 0, len(attributes))
		for k, v := range attributes {
			attrs = append(attrs, attribute.String(k, v))
		}
		span.SetAttributes(attrs...)
	}
}

// LogInfo logs an info-level message with structured data
func LogInfo(ctx context.Context, message string, attributes map[string]string) {
	if observer, ok := globalObserver.(*otelObserver); ok {
		observer.logInfo(ctx, message, attributes)
	}
}

// LogError logs an error-level message with structured data
func LogError(ctx context.Context, message string, err error, attributes map[string]string) {
	if observer, ok := globalObserver.(*otelObserver); ok {
		observer.logError(ctx, message, err, attributes)
	}
}

// noopObserver is a no-operation observer that does nothing
type noopObserver struct{}

func (n *noopObserver) OnFormValidationStart(ctx context.Context, formName string) {}
func (n *noopObserver) OnFormValidationEnd(ctx context.Context, formName string, errorCount int, duration time.Duration) {
}
func (n *noopObserver) OnFormValidationError(ctx context.Context, formName string, field string, error string) {
}
func (n *noopObserver) OnTranslationStart(ctx context.Context, locale string, key string) {}
func (n *noopObserver) OnTranslationEnd(ctx context.Context, locale string, key string, duration time.Duration) {
}
func (n *noopObserver) OnLocaleDetection(ctx context.Context, detectedLocale string, fallbackUsed bool) {
}
func (n *noopObserver) OnUploadStart(ctx context.Context, fileName string, fileSize int64) {}
func (n *noopObserver) OnUploadEnd(ctx context.Context, fileName string, fileSize int64, duration time.Duration, success bool) {
}
func (n *noopObserver) OnUploadError(ctx context.Context, fileName string, error string) {}
func (n *noopObserver) OnStorageOperation(ctx context.Context, operation string, storageType string, duration time.Duration, success bool) {
}

// otelObserver implements Observer using OpenTelemetry
type otelObserver struct {
	config Config
	tracer trace.Tracer
	meter  metric.Meter
}

func (o *otelObserver) OnFormValidationStart(ctx context.Context, formName string) {
	_, span := o.tracer.Start(ctx, "form.validation", trace.WithAttributes(
		attribute.String("form.name", formName),
	))
	span.End()
}

func (o *otelObserver) OnFormValidationEnd(ctx context.Context, formName string, errorCount int, duration time.Duration) {
	AddSpanEvent(ctx, "form.validation.completed", map[string]string{
		"form.name":   formName,
		"error.count": fmt.Sprintf("%d", errorCount),
		"duration.ms": fmt.Sprintf("%.2f", float64(duration.Microseconds())/1000.0),
	})
}

func (o *otelObserver) OnFormValidationError(ctx context.Context, formName string, field string, error string) {
	AddSpanEvent(ctx, "form.validation.error", map[string]string{
		"form.name": formName,
		"field":     field,
		"error":     error,
	})
}

func (o *otelObserver) OnTranslationStart(ctx context.Context, locale string, key string) {
	_, span := o.tracer.Start(ctx, "i18n.translation", trace.WithAttributes(
		attribute.String("locale", locale),
		attribute.String("key", key),
	))
	span.End()
}

func (o *otelObserver) OnTranslationEnd(ctx context.Context, locale string, key string, duration time.Duration) {
	AddSpanEvent(ctx, "i18n.translation.completed", map[string]string{
		"locale":      locale,
		"key":         key,
		"duration.ms": fmt.Sprintf("%.2f", float64(duration.Microseconds())/1000.0),
	})
}

func (o *otelObserver) OnLocaleDetection(ctx context.Context, detectedLocale string, fallbackUsed bool) {
	AddSpanEvent(ctx, "i18n.locale.detected", map[string]string{
		"locale":        detectedLocale,
		"fallback.used": fmt.Sprintf("%t", fallbackUsed),
	})
}

func (o *otelObserver) OnUploadStart(ctx context.Context, fileName string, fileSize int64) {
	_, span := o.tracer.Start(ctx, "upload.start", trace.WithAttributes(
		attribute.String("file.name", fileName),
		attribute.Int64("file.size", fileSize),
	))
	span.End()
}

func (o *otelObserver) OnUploadEnd(ctx context.Context, fileName string, fileSize int64, duration time.Duration, success bool) {
	AddSpanEvent(ctx, "upload.completed", map[string]string{
		"file.name":   fileName,
		"file.size":   fmt.Sprintf("%d", fileSize),
		"success":     fmt.Sprintf("%t", success),
		"duration.ms": fmt.Sprintf("%.2f", float64(duration.Microseconds())/1000.0),
	})
}

func (o *otelObserver) OnUploadError(ctx context.Context, fileName string, error string) {
	AddSpanEvent(ctx, "upload.error", map[string]string{
		"file.name": fileName,
		"error":     error,
	})
}

func (o *otelObserver) OnStorageOperation(ctx context.Context, operation string, storageType string, duration time.Duration, success bool) {
	AddSpanEvent(ctx, "storage.operation", map[string]string{
		"operation":    operation,
		"storage.type": storageType,
		"success":      fmt.Sprintf("%t", success),
		"duration.ms":  fmt.Sprintf("%.2f", float64(duration.Microseconds())/1000.0),
	})
}

func (o *otelObserver) recordMetric(name string, value float64, attributes map[string]string) {
	// For now, we'll use span events to record metrics
	// In a full implementation, you'd use the meter to create and record metrics
	ctx := context.Background()
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		attrs := make([]attribute.KeyValue, 0, len(attributes)+2)
		attrs = append(attrs, attribute.String("metric.name", name))
		attrs = append(attrs, attribute.Float64("metric.value", value))
		for k, v := range attributes {
			attrs = append(attrs, attribute.String(k, v))
		}
		span.AddEvent("metric.recorded", trace.WithAttributes(attrs...))
	}
}

func (o *otelObserver) logInfo(ctx context.Context, message string, attributes map[string]string) {
	// Use span events for logging
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		attrs := make([]attribute.KeyValue, 0, len(attributes)+1)
		attrs = append(attrs, attribute.String("message", message))
		for k, v := range attributes {
			attrs = append(attrs, attribute.String(k, v))
		}
		span.AddEvent("log.info", trace.WithAttributes(attrs...))
	}
}

func (o *otelObserver) logError(ctx context.Context, message string, err error, attributes map[string]string) {
	// Use span events for error logging
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		attrs := make([]attribute.KeyValue, 0, len(attributes)+2)
		attrs = append(attrs, attribute.String("message", message))
		if err != nil {
			attrs = append(attrs, attribute.String("error", err.Error()))
		}
		for k, v := range attributes {
			attrs = append(attrs, attribute.String(k, v))
		}
		span.AddEvent("log.error", trace.WithAttributes(attrs...))
	}
}

// initOpenTelemetry initializes OpenTelemetry with the given configuration
func initOpenTelemetry(config Config) error {
	ctx := context.Background()

	// Set up resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %v", err)
	}

	// Set up trace provider with no-op exporter for now
	// In a production environment, you would configure proper exporters
	if config.EnableTracing {
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
		)

		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
	}

	// Set up meter provider with no-op reader for now
	// In a production environment, you would configure proper readers
	if config.EnableMetrics {
		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
		)

		otel.SetMeterProvider(mp)
	}

	return nil
}
