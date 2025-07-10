package i18n

import (
	"context"
	"testing"
	"time"
)

func TestI18nObservability(t *testing.T) {
	// Test observer registration
	observer := &testI18nObserver{}
	RegisterObserver(observer)

	// Test observer retrieval
	retrieved := getObserver()
	if retrieved == nil {
		t.Error("Observer should be registered")
	}

	// Test observability hooks
	ctx := context.Background()

	observer.OnTranslationStart(ctx, "en", "test-key")
	observer.OnTranslationEnd(ctx, "en", "test-key", time.Millisecond)

	observer.OnLocaleDetection(ctx, "en", false)

	// Test observability enablement
	EnableObservability()
}

type testI18nObserver struct{}

func (o *testI18nObserver) OnTranslationStart(ctx context.Context, locale, key string) {}
func (o *testI18nObserver) OnTranslationEnd(ctx context.Context, locale, key string, duration time.Duration) {
}
func (o *testI18nObserver) OnLocaleDetection(ctx context.Context, detectedLocale string, fallbackUsed bool) {
}
