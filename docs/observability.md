# GoKit Observability

GoKit provides comprehensive observability capabilities through OpenTelemetry integration, enabling distributed tracing, metrics collection, and structured logging across all packages.

## Overview

The observability system is designed to be:
- **Zero-dependency when disabled**: No performance impact when not configured
- **Context-aware**: All operations respect context for cancellation and tracing
- **Modular**: Each package can enable observability independently
- **Standards-compliant**: Uses OpenTelemetry for industry-standard observability

## Features

### Distributed Tracing
- Automatic span creation for key operations
- Context propagation across package boundaries
- Custom span attributes and events
- Performance timing for all operations

### Metrics Collection
- Custom metrics for form validations, translations, and uploads
- Storage operation timing and success rates
- Error rate tracking
- Performance monitoring

### Structured Logging
- Context-aware logging with correlation IDs
- Structured data with key-value pairs
- Error logging with stack traces
- Integration with tracing spans

## Quick Start

### 1. Initialize Observability

```go
import "github.com/kdsmith18542/gokit/observability"

func main() {
    err := observability.Init(observability.Config{
        ServiceName:    "my-app",
        ServiceVersion: "1.0.0",
        Environment:    "production",
        EnableTracing:  true,
        EnableMetrics:  true,
        EnableLogging:  true,
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### 2. Enable Package Observability

```go
import (
    "github.com/kdsmith18542/gokit/form"
    "github.com/kdsmith18542/gokit/i18n"
    "github.com/kdsmith18542/gokit/upload"
)

// Enable observability for all packages
form.EnableObservability()
i18n.EnableObservability()
upload.EnableObservability()
```

### 3. Use in Your Application

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Start a custom span
    ctx, span := observability.StartSpan(ctx, "user_registration")
    defer span.End()
    
    // Add span attributes
    observability.SetSpanAttributes(ctx, map[string]string{
        "user_id": "12345",
        "action":  "register",
    })
    
    // Record metrics
    observability.RecordMetric("requests_total", 1, map[string]string{
        "endpoint": "/register",
        "status":   "success",
    })
    
    // Log structured events
    observability.LogInfo(ctx, "User registered", map[string]string{
        "user_email": "user@example.com",
        "method":     "form",
    })
}
```

## Package Integration

### Form Package

The form package automatically tracks:
- Form validation start/end times
- Validation error counts and details
- Field-level validation failures
- Sanitization operations

```go
// Form validation is automatically observed
errors := form.DecodeAndValidateWithContext(ctx, r, &userForm)
// Spans and metrics are automatically created
```

### i18n Package

The i18n package tracks:
- Translation operations with timing
- Locale detection and fallback usage
- Translation key lookups
- Formatting operations

```go
// Translation operations are automatically observed
message := translator.T("welcome", map[string]interface{}{
    "Name": "John",
})
// Spans and metrics are automatically created
```

### Upload Package

The upload package tracks:
- File upload start/end times
- Upload success/failure rates
- File size and type validation
- Storage operation timing

```go
// Upload operations are automatically observed
results, err := uploadProcessor.ProcessWithContext(ctx, r, "avatar")
// Spans and metrics are automatically created
```

## Storage Observability

Storage backends can be wrapped with observability:

```go
import "github.com/kdsmith18542/gokit/upload/storage"

// Create observable storage wrapper
localStorage := storage.NewLocal("./uploads")
observableStorage := storage.NewObservableStorage(localStorage, "local")

// Use with upload processor
processor := upload.NewProcessor(observableStorage, upload.Options{})
```

## Configuration Options

### Observability Config

```go
type Config struct {
    ServiceName    string // Service name for tracing
    ServiceVersion string // Service version
    Environment    string // Deployment environment
    EnableTracing  bool   // Enable distributed tracing
    EnableMetrics  bool   // Enable metrics collection
    EnableLogging  bool   // Enable structured logging
}
```

### Custom Observers

You can implement custom observers for specific needs:

```go
type CustomObserver struct{}

func (c *CustomObserver) OnFormValidationStart(ctx context.Context, formName string) {
    // Custom logic
}

func (c *CustomObserver) OnFormValidationEnd(ctx context.Context, formName string, errorCount int, duration time.Duration) {
    // Custom logic
}

// Register custom observer
form.RegisterObserver(&CustomObserver{})
```

## Advanced Usage

### Custom Spans

```go
ctx, span := observability.StartSpan(ctx, "custom_operation")
defer span.End()

// Add attributes
observability.SetSpanAttributes(ctx, map[string]string{
    "operation_type": "batch_process",
    "batch_size":     "1000",
})

// Add events
observability.AddSpanEvent(ctx, "batch_started", map[string]string{
    "timestamp": time.Now().Format(time.RFC3339),
})
```

### Custom Metrics

```go
// Record custom metrics
observability.RecordMetric("custom_operation_duration", 1.5, map[string]string{
    "operation": "data_processing",
    "status":    "success",
})
```

### Structured Logging

```go
// Log info with context
observability.LogInfo(ctx, "Operation completed", map[string]string{
    "operation": "user_creation",
    "user_id":   "12345",
    "duration":  "150ms",
})

// Log errors with context
observability.LogError(ctx, "Database connection failed", err, map[string]string{
    "database": "users",
    "host":     "db.example.com",
})
```

## Performance Considerations

### Zero Overhead When Disabled

When observability is not initialized or disabled:
- All observability calls are no-ops
- No performance impact on production code
- No external dependencies required

### Efficient When Enabled

When observability is enabled:
- Minimal overhead for span creation
- Efficient context propagation
- Optimized metric recording
- Structured logging with minimal allocations

## Integration with Monitoring Systems

### OpenTelemetry Exporters

The observability system is designed to work with OpenTelemetry exporters:

```go
// Example: Jaeger exporter
import (
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/sdk/trace"
)

func initJaeger() {
    exporter, err := jaeger.New(jaeger.WithCollectorEndpoint("http://localhost:14268/api/traces"))
    if err != nil {
        log.Fatal(err)
    }
    
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(resource.NewWithAttributes(
            semconv.ServiceNameKey.String("my-app"),
        )),
    )
    otel.SetTracerProvider(tp)
}
```

### Metrics Exporters

```go
// Example: Prometheus exporter
import (
    "go.opentelemetry.io/otel/exporters/prometheus"
    "go.opentelemetry.io/otel/sdk/metric"
)

func initPrometheus() {
    exporter, err := prometheus.New()
    if err != nil {
        log.Fatal(err)
    }
    
    mp := metric.NewMeterProvider(metric.WithReader(exporter))
    otel.SetMeterProvider(mp)
}
```

## Best Practices

### 1. Use Context Consistently

Always pass context through your application:

```go
func processRequest(ctx context.Context, r *http.Request) {
    // Use context-aware functions
    errors := form.DecodeAndValidateWithContext(ctx, r, &form)
    results, err := uploadProcessor.ProcessWithContext(ctx, r, "file")
}
```

### 2. Add Meaningful Attributes

Include relevant business context in spans:

```go
observability.SetSpanAttributes(ctx, map[string]string{
    "user_id":    userID,
    "operation":  "payment_processing",
    "amount":     fmt.Sprintf("%.2f", amount),
    "currency":   "USD",
})
```

### 3. Use Structured Logging

Avoid string concatenation in logs:

```go
// Good
observability.LogInfo(ctx, "User action", map[string]string{
    "user_id": userID,
    "action":  "login",
    "ip":      clientIP,
})

// Avoid
log.Printf("User %s performed %s from %s", userID, "login", clientIP)
```

### 4. Monitor Error Rates

Track error rates and patterns:

```go
if err != nil {
    observability.LogError(ctx, "Operation failed", err, map[string]string{
        "operation": "database_query",
        "table":     "users",
    })
    observability.RecordMetric("errors_total", 1, map[string]string{
        "type": "database_error",
    })
}
```

## Troubleshooting

### Common Issues

1. **No spans appearing**: Ensure observability is initialized and tracing is enabled
2. **Missing context**: Always pass context through function calls
3. **Performance impact**: Use sampling in production to reduce overhead
4. **Missing metrics**: Verify metrics are enabled and exporters are configured

### Debug Mode

Enable debug logging for troubleshooting:

```go
// Add debug logging to your application
if debug {
    observability.LogInfo(ctx, "Debug info", map[string]string{
        "operation": "debug_check",
        "timestamp": time.Now().Format(time.RFC3339),
    })
}
```

## Examples

See the `examples/observability-demo/` directory for a complete working example that demonstrates:

- Observability initialization
- Package integration
- Custom spans and metrics
- HTTP request handling
- Error tracking
- Performance monitoring

Run the example:

```bash
cd examples/observability-demo
go run main.go
```

Then test the endpoints:

```bash
# Health check
curl http://localhost:8080/health

# User registration (with form data)
curl -X POST http://localhost:8080/register \
  -F "email=user@example.com" \
  -F "password=password123" \
  -F "confirm_password=password123" \
  -F "name=John Doe" \
  -F "age=25"
``` 