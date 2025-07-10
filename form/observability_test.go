package form

import (
	"context"
	"testing"
)

func TestObservability(t *testing.T) {
	// Test observer registration
	observer := &testObserver{}
	RegisterObserver(observer)

	// Test observer retrieval
	retrieved := getObserver()
	if retrieved == nil {
		t.Error("Observer should be registered")
	}

	// Test observability hooks
	ctx := context.Background()

	observer.OnDecodeStart(ctx, "test-form")
	observer.OnDecodeEnd(ctx, "test-form", nil)

	observer.OnValidationStart(ctx, "test-form")
	observer.OnValidationEnd(ctx, "test-form", nil)

	// Test with errors
	observer.OnValidationEnd(ctx, "test-form", ValidationErrors{
		"field": {"error"},
	})

	// Test observability enablement
	EnableObservability()
}

type testObserver struct{}

func (o *testObserver) OnDecodeStart(ctx context.Context, formName string)          {}
func (o *testObserver) OnDecodeEnd(ctx context.Context, formName string, err error) {}
func (o *testObserver) OnValidationStart(ctx context.Context, formName string)      {}
func (o *testObserver) OnValidationEnd(ctx context.Context, formName string, errors ValidationErrors) {
}
