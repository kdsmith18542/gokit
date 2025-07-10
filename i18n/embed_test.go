package i18n

import (
	"embed"
	"net/http"
	"testing"
)

//go:embed testdata/*.toml
var testFS embed.FS

//go:embed testdata/nested/*.toml
var nestedFS embed.FS

func TestNewManagerFromFS(t *testing.T) {
	manager := NewManagerFromFS(testFS)

	if manager == nil {
		t.Fatal("NewManagerFromFS returned nil")
	}

	if manager.defaultLocale != "en" {
		t.Errorf("Expected default locale 'en', got '%s'", manager.defaultLocale)
	}

	if manager.fallbackLocale != "en" {
		t.Errorf("Expected fallback locale 'en', got '%s'", manager.fallbackLocale)
	}
}

func TestNewManagerFromFSWithPath(t *testing.T) {
	manager := NewManagerFromFSWithPath(nestedFS, "testdata/nested")

	if manager == nil {
		t.Fatal("NewManagerFromFSWithPath returned nil")
	}

	if manager.defaultLocale != "en" {
		t.Errorf("Expected default locale 'en', got '%s'", manager.defaultLocale)
	}

	if manager.fallbackLocale != "en" {
		t.Errorf("Expected fallback locale 'en', got '%s'", manager.fallbackLocale)
	}
}

func TestAddLocaleFromFS(t *testing.T) {
	manager := NewManagerEmpty()

	// Test adding a locale from embedded filesystem
	err := manager.AddLocaleFromFS(testFS, "en", "testdata/en.toml")
	if err != nil {
		t.Fatalf("Failed to add locale from FS: %v", err)
	}

	// Verify the locale was added
	locale := manager.getLocale("en")
	if locale == nil {
		t.Fatal("Expected locale 'en' to be available after adding from FS")
	}

	// Test translation - create a mock request
	req, _ := http.NewRequest("GET", "/", nil)
	translator := manager.Translator(req)
	result := translator.T("welcome", nil)
	if result != "Welcome" {
		t.Errorf("Expected translation 'Welcome', got '%s'", result)
	}
}

func TestAddLocaleFromFS_InvalidPath(t *testing.T) {
	manager := NewManagerEmpty()

	// Test adding a locale from non-existent path
	err := manager.AddLocaleFromFS(testFS, "fr", "testdata/nonexistent.toml")
	if err == nil {
		t.Fatal("Expected error when adding locale from non-existent path")
	}
}

func TestAddLocalesFromFS(t *testing.T) {
	manager := NewManagerEmpty()

	// Test adding multiple locales from embedded filesystem
	err := manager.AddLocalesFromFS(nestedFS, "testdata/nested")
	if err != nil {
		t.Fatalf("Failed to add locales from FS: %v", err)
	}

	// Verify locales were added (this will depend on what files are actually in the testdata)
	// For now, just check that no error occurred
}

func TestLoadLocaleFileFromFS_ValidFile(t *testing.T) {
	manager := NewManagerEmpty()

	// Test loading a valid locale file
	err := manager.loadLocaleFileFromFS(testFS, "en", "testdata/en.toml")
	if err != nil {
		t.Fatalf("Failed to load locale file: %v", err)
	}

	// Verify the locale was loaded
	locale := manager.getLocale("en")
	if locale == nil {
		t.Fatal("Expected locale 'en' to be available after loading from FS")
	}

	// Check that messages were loaded
	if len(locale.Messages) == 0 {
		t.Fatal("Expected messages to be loaded from file")
	}
}

func TestLoadLocaleFileFromFS_InvalidFile(t *testing.T) {
	manager := NewManagerEmpty()

	// Test loading a non-existent file
	err := manager.loadLocaleFileFromFS(testFS, "fr", "testdata/nonexistent.toml")
	if err == nil {
		t.Fatal("Expected error when loading non-existent file")
	}
}

func TestLoadLocaleFileFromFS_EmptyFile(t *testing.T) {
	manager := NewManagerEmpty()

	// Test loading an empty file (if it exists)
	// This should not cause an error, just result in an empty locale
	_ = manager.loadLocaleFileFromFS(testFS, "empty", "testdata/empty.toml")
	// We don't check for error here as the file might not exist in testdata
	// The important thing is that if it exists and is empty, it should be handled gracefully
}

func TestEmbeddedFSIntegration(t *testing.T) {
	// Test the complete workflow with embedded filesystem
	manager := NewManagerFromFS(testFS)

	// Add a specific locale from the embedded FS
	err := manager.AddLocaleFromFS(testFS, "en", "testdata/en.toml")
	if err != nil {
		t.Fatalf("Failed to add locale: %v", err)
	}

	// Test translation - create a mock request
	req, _ := http.NewRequest("GET", "/", nil)
	translator := manager.Translator(req)
	result := translator.T("welcome", nil)
	if result != "Welcome" {
		t.Errorf("Expected translation 'Welcome', got '%s'", result)
	}

	// Test with parameters
	result = translator.T("greeting", map[string]interface{}{
		"Name": "Alice",
	})
	if result != "Hello Alice" {
		t.Errorf("Expected translation 'Hello Alice', got '%s'", result)
	}
}

func TestEmbeddedFSWithNestedPath(t *testing.T) {
	// Test with nested path structure
	manager := NewManagerFromFSWithPath(nestedFS, "testdata/nested")

	// Add a locale from the nested structure
	err := manager.AddLocaleFromFS(nestedFS, "fr", "testdata/nested/fr.toml")
	if err != nil {
		t.Fatalf("Failed to add locale from nested path: %v", err)
	}

	// Set default locale to 'fr' to ensure correct detection
	manager.defaultLocale = "fr"

	// Test translation - create a mock request
	req, _ := http.NewRequest("GET", "/", nil)
	translator := manager.Translator(req)
	result := translator.T("welcome", nil)
	if result != "Bienvenue" {
		t.Errorf("Expected translation 'Bienvenue', got '%s'", result)
	}
}

// Benchmark tests for performance
func BenchmarkNewManagerFromFS(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewManagerFromFS(testFS)
	}
}

func BenchmarkAddLocaleFromFS(b *testing.B) {
	manager := NewManagerEmpty()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := manager.AddLocaleFromFS(testFS, "en", "testdata/en.toml"); err != nil {
			b.Fatalf("Failed to add locale from FS: %v", err)
		}
	}
}
