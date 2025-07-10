package upload

import (
	"context"
	"time"

	"github.com/kdsmith18542/gokit/observability"
)

// Observer defines hooks for tracing and metrics in upload operations
type Observer interface {
	OnUploadStart(ctx context.Context, fileName string, fileSize int64)
	OnUploadEnd(ctx context.Context, fileName string, fileSize int64, duration time.Duration, success bool)
	OnUploadError(ctx context.Context, fileName string, error string)
	OnStorageOperation(ctx context.Context, operation string, storageType string, duration time.Duration, success bool)
}

var observer Observer

// RegisterObserver sets the global observer for upload events
func RegisterObserver(obs Observer) {
	observer = obs
}

// getObserver returns the registered observer (or nil)
func getObserver() Observer {
	return observer
}

// uploadObserver implements Observer using the global observability system
type uploadObserver struct{}

func (u *uploadObserver) OnUploadStart(ctx context.Context, fileName string, fileSize int64) {
	observability.GetObserver().OnUploadStart(ctx, fileName, fileSize)
}

func (u *uploadObserver) OnUploadEnd(ctx context.Context, fileName string, fileSize int64, duration time.Duration, success bool) {
	observability.GetObserver().OnUploadEnd(ctx, fileName, fileSize, duration, success)
}

func (u *uploadObserver) OnUploadError(ctx context.Context, fileName string, error string) {
	observability.GetObserver().OnUploadError(ctx, fileName, error)
}

func (u *uploadObserver) OnStorageOperation(ctx context.Context, operation string, storageType string, duration time.Duration, success bool) {
	observability.GetObserver().OnStorageOperation(ctx, operation, storageType, duration, success)
}

// EnableObservability enables observability integration for the upload package
func EnableObservability() {
	RegisterObserver(&uploadObserver{})
}
