package upload

import (
	"context"
	"testing"
	"time"
)

func TestUploadObservability(t *testing.T) {
	// Test observer registration
	observer := &testUploadObserver{}
	RegisterObserver(observer)

	// Test observer retrieval
	retrieved := getObserver()
	if retrieved == nil {
		t.Error("Observer should be registered")
	}

	// Test observability hooks
	ctx := context.Background()

	observer.OnUploadStart(ctx, "test-file.txt", 1024)
	observer.OnUploadEnd(ctx, "test-file.txt", 1024, time.Millisecond, true)

	observer.OnUploadError(ctx, "test-file.txt", "test error")
	observer.OnStorageOperation(ctx, "store", "local", time.Millisecond, true)

	// Test observability enablement
	EnableObservability()
}

type testUploadObserver struct{}

func (o *testUploadObserver) OnUploadStart(ctx context.Context, fileName string, fileSize int64) {}
func (o *testUploadObserver) OnUploadEnd(ctx context.Context, fileName string, fileSize int64, duration time.Duration, success bool) {
}
func (o *testUploadObserver) OnUploadError(ctx context.Context, fileName string, error string) {}
func (o *testUploadObserver) OnStorageOperation(ctx context.Context, operation string, storageType string, duration time.Duration, success bool) {
}
