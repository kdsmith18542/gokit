package observability

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TestObservability_Init(t *testing.T) {
	// Test initialization with no observability enabled
	err := Init(Config{})
	if err != nil {
		t.Errorf("Expected no error when no observability enabled, got: %v", err)
	}

	// Test initialization with tracing enabled
	err = Init(Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		EnableTracing:  true,
	})
	if err != nil {
		t.Errorf("Expected no error when tracing enabled, got: %v", err)
	}

	// Test initialization with metrics enabled
	err = Init(Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		EnableMetrics:  true,
	})
	if err != nil {
		t.Errorf("Expected no error when metrics enabled, got: %v", err)
	}

	// Test initialization with logging enabled
	err = Init(Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		EnableLogging:  true,
	})
	if err != nil {
		t.Errorf("Expected no error when logging enabled, got: %v", err)
	}

	// Test initialization with all features enabled
	err = Init(Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		EnableTracing:  true,
		EnableMetrics:  true,
		EnableLogging:  true,
	})
	if err != nil {
		t.Errorf("Expected no error when all features enabled, got: %v", err)
	}
}

func TestObservability_SetAndGetObserver(t *testing.T) {
	// Test default observer
	observer := GetObserver()
	if observer == nil {
		t.Error("Expected default observer to not be nil")
	}

	// Test setting custom observer
	customObserver := &testObserver{}
	SetObserver(customObserver)

	// Verify observer was set
	currentObserver := GetObserver()
	if currentObserver != customObserver {
		t.Error("Expected observer to be set to custom observer")
	}

	// Test observer methods
	ctx := context.Background()
	customObserver.OnFormValidationStart(ctx, "test-form")
	customObserver.OnFormValidationEnd(ctx, "test-form", 0, time.Millisecond)
	customObserver.OnFormValidationError(ctx, "test-form", "email", "invalid email")
	customObserver.OnTranslationStart(ctx, "en", "welcome")
	customObserver.OnTranslationEnd(ctx, "en", "welcome", time.Millisecond)
	customObserver.OnLocaleDetection(ctx, "en", false)
	customObserver.OnUploadStart(ctx, "test.txt", 1024)
	customObserver.OnUploadEnd(ctx, "test.txt", 1024, time.Millisecond, true)
	customObserver.OnUploadError(ctx, "test.txt", "upload failed")
	customObserver.OnStorageOperation(ctx, "store", "local", time.Millisecond, true)
}

func TestObservability_StartSpan(t *testing.T) {
	ctx := context.Background()

	// Test starting a span
	spanCtx, span := StartSpan(ctx, "test-operation")
	if spanCtx == nil {
		t.Error("Expected span context to not be nil")
	}
	if span == nil {
		t.Error("Expected span to not be nil")
	}
	span.End()

	// Test starting a span with options
	spanCtx, span = StartSpan(ctx, "test-operation-with-options", trace.WithAttributes(
		attribute.String("test.key", "test.value"),
	))
	if spanCtx == nil {
		t.Error("Expected span context to not be nil")
	}
	if span == nil {
		t.Error("Expected span to not be nil")
	}
	span.End()
}

func TestObservability_RecordMetric(t *testing.T) {
	// Test recording metrics
	RecordMetric("test_metric", 1.0, map[string]string{
		"test": "value",
	})

	// Test recording metrics with empty attributes
	RecordMetric("test_metric_empty", 2.0, nil)

	// Test recording metrics with multiple attributes
	RecordMetric("test_metric_multi", 3.0, map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	})
}

func TestObservability_AddSpanEvent(t *testing.T) {
	ctx := context.Background()

	// Test adding span event (should not panic even without active span)
	AddSpanEvent(ctx, "test-event", map[string]string{
		"test": "value",
	})

	// Test adding span event with empty attributes
	AddSpanEvent(ctx, "test-event-empty", nil)

	// Test adding span event with multiple attributes
	AddSpanEvent(ctx, "test-event-multi", map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	})

	// Test with actual span context
	spanCtx, span := StartSpan(ctx, "test-span")
	defer span.End()

	AddSpanEvent(spanCtx, "test-event-with-span", map[string]string{
		"test": "value",
	})
}

func TestObservability_SetSpanAttributes(t *testing.T) {
	ctx := context.Background()

	// Test setting span attributes (should not panic even without active span)
	SetSpanAttributes(ctx, map[string]string{
		"test": "value",
	})

	// Test setting span attributes with empty map
	SetSpanAttributes(ctx, nil)

	// Test setting span attributes with multiple attributes
	SetSpanAttributes(ctx, map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	})

	// Test with actual span context
	spanCtx, span := StartSpan(ctx, "test-span")
	defer span.End()

	SetSpanAttributes(spanCtx, map[string]string{
		"test": "value",
	})
}

func TestObservability_LogInfo(t *testing.T) {
	ctx := context.Background()

	// Test logging info
	LogInfo(ctx, "test info message", map[string]string{
		"test": "value",
	})

	// Test logging info with empty attributes
	LogInfo(ctx, "test info message empty", nil)

	// Test logging info with multiple attributes
	LogInfo(ctx, "test info message multi", map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	})
}

func TestObservability_LogError(t *testing.T) {
	ctx := context.Background()
	testErr := errors.New("test error")

	// Test logging error
	LogError(ctx, "test error message", testErr, map[string]string{
		"test": "value",
	})

	// Test logging error with empty attributes
	LogError(ctx, "test error message empty", testErr, nil)

	// Test logging error with multiple attributes
	LogError(ctx, "test error message multi", testErr, map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	})

	// Test logging error with nil error
	LogError(ctx, "test error message nil", nil, map[string]string{
		"test": "value",
	})
}

func TestObservability_ObserverImplementation(t *testing.T) {
	// Test noop observer
	noop := &noopObserver{}
	ctx := context.Background()

	// These should not panic
	noop.OnFormValidationStart(ctx, "test-form")
	noop.OnFormValidationEnd(ctx, "test-form", 0, time.Millisecond)
	noop.OnFormValidationError(ctx, "test-form", "email", "invalid email")
	noop.OnTranslationStart(ctx, "en", "welcome")
	noop.OnTranslationEnd(ctx, "en", "welcome", time.Millisecond)
	noop.OnLocaleDetection(ctx, "en", false)
	noop.OnUploadStart(ctx, "test.txt", 1024)
	noop.OnUploadEnd(ctx, "test.txt", 1024, time.Millisecond, true)
	noop.OnUploadError(ctx, "test.txt", "upload failed")
	noop.OnStorageOperation(ctx, "store", "local", time.Millisecond, true)
}

func TestObservability_PrivateMethods(t *testing.T) {
	// Test otelObserver private methods
	observer := &otelObserver{
		config: Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
		},
		tracer: otel.Tracer("gokit"),
		meter:  otel.Meter("gokit"),
	}

	ctx := context.Background()

	// Test recordMetric
	observer.recordMetric("test_metric", 1.0, map[string]string{
		"test": "value",
	})

	// Test logInfo
	observer.logInfo(ctx, "test info", map[string]string{
		"test": "value",
	})

	// Test logError
	testErr := errors.New("test error")
	observer.logError(ctx, "test error", testErr, map[string]string{
		"test": "value",
	})

	// Test logError with nil error
	observer.logError(ctx, "test error nil", nil, map[string]string{
		"test": "value",
	})
}

func TestObservability_UploadObserverMethods(t *testing.T) {
	observer := &otelObserver{
		config: Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
		},
		tracer: otel.Tracer("gokit"),
		meter:  otel.Meter("gokit"),
	}

	ctx := context.Background()

	// Test all upload observer methods
	observer.OnUploadStart(ctx, "test.txt", 1024)
	observer.OnUploadEnd(ctx, "test.txt", 1024, time.Millisecond, true)
	observer.OnUploadEnd(ctx, "test.txt", 1024, time.Millisecond, false)
	observer.OnUploadError(ctx, "test.txt", "upload failed")
	observer.OnStorageOperation(ctx, "store", "local", time.Millisecond, true)
	observer.OnStorageOperation(ctx, "delete", "s3", time.Millisecond, false)
}

func TestObservability_FormObserverMethods(t *testing.T) {
	observer := &otelObserver{
		config: Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
		},
		tracer: otel.Tracer("gokit"),
		meter:  otel.Meter("gokit"),
	}

	ctx := context.Background()

	// Test all form observer methods
	observer.OnFormValidationStart(ctx, "registration-form")
	observer.OnFormValidationEnd(ctx, "registration-form", 0, time.Millisecond)
	observer.OnFormValidationEnd(ctx, "registration-form", 3, time.Millisecond)
	observer.OnFormValidationError(ctx, "registration-form", "email", "invalid email")
}

func TestObservability_I18nObserverMethods(t *testing.T) {
	observer := &otelObserver{
		config: Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
		},
		tracer: otel.Tracer("gokit"),
		meter:  otel.Meter("gokit"),
	}

	ctx := context.Background()

	// Test all i18n observer methods
	observer.OnTranslationStart(ctx, "en", "welcome")
	observer.OnTranslationEnd(ctx, "en", "welcome", time.Millisecond)
	observer.OnLocaleDetection(ctx, "en", false)
	observer.OnLocaleDetection(ctx, "en", true)
}

func TestObservability_EdgeCases(t *testing.T) {
	ctx := context.Background()

	// Test with empty strings
	RecordMetric("", 0.0, nil)
	AddSpanEvent(ctx, "", nil)
	SetSpanAttributes(ctx, nil)
	LogInfo(ctx, "", nil)
	LogError(ctx, "", nil, nil)

	// Test with very long strings
	longString := string(make([]byte, 10000))
	RecordMetric(longString, 0.0, map[string]string{longString: longString})
	AddSpanEvent(ctx, longString, map[string]string{longString: longString})
	SetSpanAttributes(ctx, map[string]string{longString: longString})
	LogInfo(ctx, longString, map[string]string{longString: longString})
	LogError(ctx, longString, errors.New(longString), map[string]string{longString: longString})

	// Test with special characters
	specialChars := "!@#$%^&*()_+-=[]{}|;':\",./<>?"
	RecordMetric(specialChars, 0.0, map[string]string{specialChars: specialChars})
	AddSpanEvent(ctx, specialChars, map[string]string{specialChars: specialChars})
	SetSpanAttributes(ctx, map[string]string{specialChars: specialChars})
	LogInfo(ctx, specialChars, map[string]string{specialChars: specialChars})
	LogError(ctx, specialChars, errors.New(specialChars), map[string]string{specialChars: specialChars})
}

// testObserver is a test implementation of Observer for testing
type testObserver struct {
	formValidationStartCount int
	formValidationEndCount   int
	formValidationErrorCount int
	translationStartCount    int
	translationEndCount      int
	localeDetectionCount     int
	uploadStartCount         int
	uploadEndCount           int
	uploadErrorCount         int
	storageOperationCount    int
}

func (t *testObserver) OnFormValidationStart(ctx context.Context, formName string) {
	t.formValidationStartCount++
}

func (t *testObserver) OnFormValidationEnd(ctx context.Context, formName string, errorCount int, duration time.Duration) {
	t.formValidationEndCount++
}

func (t *testObserver) OnFormValidationError(ctx context.Context, formName string, field string, error string) {
	t.formValidationErrorCount++
}

func (t *testObserver) OnTranslationStart(ctx context.Context, locale string, key string) {
	t.translationStartCount++
}

func (t *testObserver) OnTranslationEnd(ctx context.Context, locale string, key string, duration time.Duration) {
	t.translationEndCount++
}

func (t *testObserver) OnLocaleDetection(ctx context.Context, detectedLocale string, fallbackUsed bool) {
	t.localeDetectionCount++
}

func (t *testObserver) OnUploadStart(ctx context.Context, fileName string, fileSize int64) {
	t.uploadStartCount++
}

func (t *testObserver) OnUploadEnd(ctx context.Context, fileName string, fileSize int64, duration time.Duration, success bool) {
	t.uploadEndCount++
}

func (t *testObserver) OnUploadError(ctx context.Context, fileName string, error string) {
	t.uploadErrorCount++
}

func (t *testObserver) OnStorageOperation(ctx context.Context, operation string, storageType string, duration time.Duration, success bool) {
	t.storageOperationCount++
}
