package i18n

import (
	"context"
	"time"

	"github.com/kdsmith18542/gokit/observability"
)

// Observer defines hooks for tracing and metrics in i18n operations
type Observer interface {
	OnTranslationStart(ctx context.Context, locale string, key string)
	OnTranslationEnd(ctx context.Context, locale string, key string, duration time.Duration)
	OnLocaleDetection(ctx context.Context, detectedLocale string, fallbackUsed bool)
}

var observer Observer

// RegisterObserver sets the global observer for i18n events
func RegisterObserver(obs Observer) {
	observer = obs
}

// getObserver returns the registered observer (or nil)
func getObserver() Observer {
	return observer
}

// i18nObserver implements Observer using the global observability system
type i18nObserver struct{}

func (i *i18nObserver) OnTranslationStart(ctx context.Context, locale string, key string) {
	observability.GetObserver().OnTranslationStart(ctx, locale, key)
}

func (i *i18nObserver) OnTranslationEnd(ctx context.Context, locale string, key string, duration time.Duration) {
	observability.GetObserver().OnTranslationEnd(ctx, locale, key, duration)
}

func (i *i18nObserver) OnLocaleDetection(ctx context.Context, detectedLocale string, fallbackUsed bool) {
	observability.GetObserver().OnLocaleDetection(ctx, detectedLocale, fallbackUsed)
}

// EnableObservability enables observability integration for the i18n package
func EnableObservability() {
	RegisterObserver(&i18nObserver{})
}
