package form

import (
	"context"

	"github.com/kdsmith18542/gokit/observability"
)

// Observer defines hooks for tracing and metrics
// Users can implement this interface and register with RegisterObserver
// All hooks are optional (no-op if not set)
type Observer interface {
	OnDecodeStart(ctx context.Context, formName string)
	OnDecodeEnd(ctx context.Context, formName string, err error)
	OnValidationStart(ctx context.Context, formName string)
	OnValidationEnd(ctx context.Context, formName string, errors ValidationErrors)
}

var observer Observer

// RegisterObserver sets the global observer for form events
func RegisterObserver(obs Observer) {
	observer = obs
}

// getObserver returns the registered observer (or nil)
func getObserver() Observer {
	return observer
}

// formObserver implements Observer using the global observability system
type formObserver struct{}

func (f *formObserver) OnDecodeStart(ctx context.Context, formName string) {
	observability.GetObserver().OnFormValidationStart(ctx, formName)
}

func (f *formObserver) OnDecodeEnd(ctx context.Context, formName string, err error) {
	// This is called at the end of the entire validation process
	// The actual timing and error count are handled in the main validation function
}

func (f *formObserver) OnValidationStart(ctx context.Context, formName string) {
	// Validation starts immediately after decode
}

func (f *formObserver) OnValidationEnd(ctx context.Context, formName string, errors ValidationErrors) {
	errorCount := 0
	for _, fieldErrors := range errors {
		errorCount += len(fieldErrors)
	}

	// Record validation completion
	observability.GetObserver().OnFormValidationEnd(ctx, formName, errorCount, 0) // Duration will be set by caller

	// Record individual validation errors
	for field, fieldErrors := range errors {
		for _, err := range fieldErrors {
			observability.GetObserver().OnFormValidationError(ctx, formName, field, err)
		}
	}
}

// EnableObservability enables observability integration for the form package
func EnableObservability() {
	RegisterObserver(&formObserver{})
}
