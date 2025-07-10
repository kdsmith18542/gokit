package i18n

import (
	"testing"
)

func TestWatchLocalesWithNonExistentDirectory(t *testing.T) {
	manager := NewManagerEmpty()

	// Test with non-existent directory
	err := manager.WatchLocales("/non/existent/path")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}

func TestWatchLocalesWithEmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManagerEmpty()

	// Test with empty directory
	err := manager.WatchLocales(tempDir)
	if err != nil {
		t.Fatalf("WatchLocales failed: %v", err)
	}

	// Should not cause any errors - minimal wait
	// Note: We don't test file modification to avoid hanging tests
}
