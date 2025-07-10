package editor

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestLocales(t *testing.T) string {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "en.toml"), []byte("hello = \"Hello\"\ngoodbye = \"Goodbye\"\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "es.toml"), []byte("hello = \"Hola\"\ngoodbye = \"Adiós\"\n"), 0644)
	return dir
}

func TestLocalesAPI(t *testing.T) {
	dir := setupTestLocales(t)
	h := NewHandler(EditorConfig{LocalesDir: dir, Manager: nil})
	r := httptest.NewRequest("GET", "/api/locales", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "en") || !strings.Contains(body, "es") {
		t.Errorf("Expected locales in response, got: %s", body)
	}
}

func TestTranslationsAPI(t *testing.T) {
	dir := setupTestLocales(t)
	h := NewHandler(EditorConfig{LocalesDir: dir, Manager: nil})
	r := httptest.NewRequest("GET", "/api/translations", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var data TranslationData
	if err := json.NewDecoder(w.Body).Decode(&data); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	// Check that we have the expected locales
	if len(data.Locales) != 2 {
		t.Errorf("Expected 2 locales, got %d", len(data.Locales))
	}

	// Check that we have the expected keys
	if len(data.Keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(data.Keys))
	}

	// Check that we have messages for each locale
	if len(data.Messages["en"]) != 2 {
		t.Errorf("Expected 2 messages for en, got %d", len(data.Messages["en"]))
	}

	if len(data.Messages["es"]) != 2 {
		t.Errorf("Expected 2 messages for es, got %d", len(data.Messages["es"]))
	}

	// Check specific message content
	if data.Messages["en"]["hello"] != "Hello" {
		t.Errorf("Expected 'Hello' for en.hello, got '%s'", data.Messages["en"]["hello"])
	}

	if data.Messages["es"]["hello"] != "Hola" {
		t.Errorf("Expected 'Hola' for es.hello, got '%s'", data.Messages["es"]["hello"])
	}
}

func TestSaveAPI(t *testing.T) {
	dir := setupTestLocales(t)
	h := NewHandler(EditorConfig{LocalesDir: dir, Manager: nil})

	// Create test data
	testData := TranslationData{
		Keys:    []string{"hello", "goodbye"},
		Locales: []string{"en", "es"},
		Messages: map[string]map[string]string{
			"en": {
				"hello":   "Hi",
				"goodbye": "Bye",
			},
			"es": {
				"hello":   "Hola",
				"goodbye": "Adiós",
			},
		},
	}

	jsonData, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	r := httptest.NewRequest("POST", "/api/save", strings.NewReader(string(jsonData)))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	// Check that files were updated
	enContent, err := os.ReadFile(filepath.Join(dir, "en.toml"))
	if err != nil {
		t.Fatalf("Failed to read en.toml: %v", err)
	}

	if !strings.Contains(string(enContent), "hello = \"Hi\"") {
		t.Errorf("Expected updated content in en.toml, got: %s", string(enContent))
	}

	esContent, err := os.ReadFile(filepath.Join(dir, "es.toml"))
	if err != nil {
		t.Fatalf("Failed to read es.toml: %v", err)
	}

	if !strings.Contains(string(esContent), "hello = \"Hola\"") {
		t.Errorf("Expected updated content in es.toml, got: %s", string(esContent))
	}
}

func TestEditorUI(t *testing.T) {
	dir := setupTestLocales(t)
	h := NewHandler(EditorConfig{LocalesDir: dir, Manager: nil})
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "GoKit i18n Editor") {
		t.Errorf("Expected editor title in response, got: %s", body[:200])
	}

	if !strings.Contains(body, "translation-table") {
		t.Errorf("Expected table structure in response")
	}
}
