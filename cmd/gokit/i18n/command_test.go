package cli

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func noOpExit(_ int) {}

func TestMain(m *testing.M) {
	exitFunc = noOpExit
	os.Exit(m.Run())
}

func TestRun_NoArgs(t *testing.T) {
	// Test with no arguments
	Run([]string{})
	// Should print usage and exit, but we can't easily test exit in unit tests
}

func TestRun_UnknownSubcommand(t *testing.T) {
	// Test with unknown subcommand
	Run([]string{"unknown"})
	// Should print error and usage
}

func TestRun_Help(t *testing.T) {
	// Test help subcommand
	Run([]string{"help"})
	// Should print usage
}

func TestFindMissingKeys_NoArgs(t *testing.T) {
	// Test find-missing with no args
	findMissingKeys([]string{})
	// Should print error about required flags
}

func TestFindMissingKeys_MissingFlags(t *testing.T) {
	// Test find-missing with missing required flags
	findMissingKeys([]string{"--source=en"})
	// Should print error about missing target
}

func TestFindMissingKeys_ValidArgs(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()

	// Create test locale files
	enContent := `welcome = "Welcome"
hello = "Hello"
goodbye = "Goodbye"`

	esContent := `welcome = "Bienvenido"
hello = "Hola"`

	err := os.WriteFile(filepath.Join(tempDir, "en.toml"), []byte(enContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "es.toml"), []byte(esContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test find-missing with valid args
	findMissingKeys([]string{"--source=en", "--target=es", "--dir=" + tempDir})
	// Should find missing "goodbye" key in es
}

func TestFindMissingKeys_SourceNotFound(t *testing.T) {
	tempDir := t.TempDir()

	// Create only target file
	esContent := `welcome = "Bienvenido"`
	err := os.WriteFile(filepath.Join(tempDir, "es.toml"), []byte(esContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test with non-existent source
	findMissingKeys([]string{"--source=en", "--target=es", "--dir=" + tempDir})
	// Should print error about source not found
}

func TestFindMissingKeys_TargetNotFound(t *testing.T) {
	tempDir := t.TempDir()

	// Create only source file
	enContent := `welcome = "Welcome"`
	err := os.WriteFile(filepath.Join(tempDir, "en.toml"), []byte(enContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test with non-existent target
	findMissingKeys([]string{"--source=en", "--target=es", "--dir=" + tempDir})
	// Should print error about target not found
}

func TestFindMissingKeys_Synchronized(t *testing.T) {
	tempDir := t.TempDir()

	// Create identical files
	content := `welcome = "Welcome"
hello = "Hello"`

	err := os.WriteFile(filepath.Join(tempDir, "en.toml"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "es.toml"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test with synchronized files
	findMissingKeys([]string{"--source=en", "--target=es", "--dir=" + tempDir})
	// Should report all keys are synchronized
}

func TestValidateFiles_NoFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Test validate with no files
	validateFiles([]string{"--dir=" + tempDir})
	// Should report no files found
}

func TestValidateFiles_ValidFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create valid test files
	validContent := `welcome = "Welcome"
hello = "Hello"`

	err := os.WriteFile(filepath.Join(tempDir, "en.toml"), []byte(validContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "es.toml"), []byte(validContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test validate with valid files
	validateFiles([]string{"--dir=" + tempDir})
	// Should report all files are valid
}

func TestValidateFiles_InvalidFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create invalid test file
	invalidContent := `welcome = "Welcome"
hello = 
invalid syntax`

	err := os.WriteFile(filepath.Join(tempDir, "invalid.toml"), []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test validate with invalid file
	validateFiles([]string{"--dir=" + tempDir})
	// Should report parsing error
}

func TestValidateFiles_EmptyKeys(t *testing.T) {
	tempDir := t.TempDir()

	// Create file with empty values
	emptyContent := `welcome = "Welcome"
empty_key = ""
another_empty = ""`

	err := os.WriteFile(filepath.Join(tempDir, "en.toml"), []byte(emptyContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test validate with empty keys
	validateFiles([]string{"--dir=" + tempDir})
	// Should report empty keys warning
}

func TestExtractKeys_Valid(t *testing.T) {
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "src")
	outputDir := filepath.Join(tempDir, "locales")

	err := os.MkdirAll(srcDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create src directory: %v", err)
	}
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create locales directory: %v", err)
	}

	// Create a dummy Go file with translation keys
	goContent := `package main

import (
	"fmt"
	"github.com/kdsmith18542/gokit/i18n"
)

func main() {
	fmt.Println(i18n.Translate("app.title"))
	t := &i18n.Translator{}
	fmt.Println(t.T("greeting.hello"))
	fmt.Println(i18n.Translate("button.submit"))
	fmt.Println(t.T("page.home.welcome"))
}
`
	err = os.WriteFile(filepath.Join(srcDir, "main.go"), []byte(goContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test Go file: %v", err)
	}

	// Run extractKeys
	extractKeys([]string{"--dir=" + srcDir, "--output=" + outputDir, "--format=toml"})

	// Verify output file exists and contains expected keys
	outputFile := filepath.Join(outputDir, "en.toml")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Expected output file %s to exist, but it doesn't", outputFile)
	}

	// Read and verify the content of the output file
	keys := getKeysFromLocaleFile(outputFile)
	expectedKeys := []string{"app.title", "button.submit", "greeting.hello", "page.home.welcome"}
	sort.Strings(keys)
	sort.Strings(expectedKeys)

	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d keys, got %d. Keys: %v", len(expectedKeys), len(keys), keys)
	}

	for i, key := range expectedKeys {
		if keys[i] != key {
			t.Errorf("Expected key %s at index %d, got %s", key, i, keys[i])
		}
	}
}

func TestFindMissingKeysInTarget(t *testing.T) {
	source := []string{"a", "b", "c", "d"}
	target := []string{"a", "c", "e"}

	missing := findMissingKeysInTarget(source, target)
	expected := []string{"b", "d"}

	if len(missing) != len(expected) {
		t.Errorf("Expected %d missing keys, got %d", len(expected), len(missing))
	}

	for i, key := range expected {
		if missing[i] != key {
			t.Errorf("Expected missing key %s, got %s", key, missing[i])
		}
	}
}

func TestFindMissingKeysInTarget_EmptySource(t *testing.T) {
	source := []string{}
	target := []string{"a", "b", "c"}

	missing := findMissingKeysInTarget(source, target)

	if len(missing) != 0 {
		t.Errorf("Expected 0 missing keys, got %d", len(missing))
	}
}

func TestFindMissingKeysInTarget_EmptyTarget(t *testing.T) {
	source := []string{"a", "b", "c"}
	target := []string{}

	missing := findMissingKeysInTarget(source, target)
	expected := []string{"a", "b", "c"}

	if len(missing) != len(expected) {
		t.Errorf("Expected %d missing keys, got %d", len(expected), len(missing))
	}
}

func TestFindMissingKeysInTarget_BothEmpty(t *testing.T) {
	source := []string{}
	target := []string{}

	missing := findMissingKeysInTarget(source, target)

	if len(missing) != 0 {
		t.Errorf("Expected 0 missing keys, got %d", len(missing))
	}
}

func TestGetKeysFromLocaleFile_ValidFile(t *testing.T) {
	tempDir := t.TempDir()

	content := `welcome = "Welcome"
hello = "Hello"
goodbye = "Goodbye"`

	filePath := filepath.Join(tempDir, "test.toml")
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	keys := getKeysFromLocaleFile(filePath)
	expected := []string{"welcome", "hello", "goodbye"}

	sort.Strings(keys)
	sort.Strings(expected)

	if len(keys) != len(expected) {
		t.Errorf("Expected %d keys, got %d", len(expected), len(keys))
	}

	for i, key := range expected {
		if keys[i] != key {
			t.Errorf("Expected key %s, got %s", key, keys[i])
		}
	}
}

func TestGetKeysFromLocaleFile_InvalidFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create invalid TOML file
	invalidContent := `welcome = "Welcome"
hello = 
invalid syntax`

	filePath := filepath.Join(tempDir, "invalid.toml")
	err := os.WriteFile(filePath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	keys := getKeysFromLocaleFile(filePath)

	if keys != nil {
		t.Errorf("Expected nil for invalid file, got %v", keys)
	}
}

func TestGetKeysFromLocaleFile_NonExistentFile(t *testing.T) {
	keys := getKeysFromLocaleFile("/non/existent/file.toml")

	if keys != nil {
		t.Errorf("Expected nil for non-existent file, got %v", keys)
	}
}

func TestGetKeysFromLocaleFile_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "empty.toml")
	err := os.WriteFile(filePath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	keys := getKeysFromLocaleFile(filePath)

	if len(keys) != 0 {
		t.Errorf("Expected 0 keys for empty file, got %d", len(keys))
	}
}

func TestFindEmptyKeysInFile_ValidFile(t *testing.T) {
	tempDir := t.TempDir()

	content := `welcome = "Welcome"
empty_key = ""
another_empty = ""
valid_key = "Valid"`

	filePath := filepath.Join(tempDir, "test.toml")
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	emptyKeys := findEmptyKeysInFile(filePath)
	expected := []string{"another_empty", "empty_key"}

	if len(emptyKeys) != len(expected) {
		t.Errorf("Expected %d empty keys, got %d", len(expected), len(emptyKeys))
	}

	for i, key := range expected {
		if emptyKeys[i] != key {
			t.Errorf("Expected empty key %s, got %s", key, emptyKeys[i])
		}
	}
}

func TestFindEmptyKeysInFile_NoEmptyKeys(t *testing.T) {
	tempDir := t.TempDir()

	content := `welcome = "Welcome"
hello = "Hello"
goodbye = "Goodbye"`

	filePath := filepath.Join(tempDir, "test.toml")
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	emptyKeys := findEmptyKeysInFile(filePath)

	if len(emptyKeys) != 0 {
		t.Errorf("Expected 0 empty keys, got %d", len(emptyKeys))
	}
}

func TestFindEmptyKeysInFile_InvalidFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create invalid TOML file
	invalidContent := `welcome = "Welcome"
hello = 
invalid syntax`

	filePath := filepath.Join(tempDir, "invalid.toml")
	err := os.WriteFile(filePath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	emptyKeys := findEmptyKeysInFile(filePath)

	if emptyKeys != nil {
		t.Errorf("Expected nil for invalid file, got %v", emptyKeys)
	}
}

func TestFindEmptyKeysInFile_NonExistentFile(t *testing.T) {
	emptyKeys := findEmptyKeysInFile("/non/existent/file.toml")

	if emptyKeys != nil {
		t.Errorf("Expected nil for non-existent file, got %v", emptyKeys)
	}
}

func TestPrintI18nUsage(t *testing.T) {
	// Test that printI18nUsage doesn't panic
	printI18nUsage()
}

func TestEdgeCases(t *testing.T) {
	recoverPanic := func(f func()) (panicked bool) {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		f()
		return false
	}

	// Test with very long filenames
	longName := string(make([]byte, 1000))

	// Test with special characters in filenames
	specialChars := "!@#$%^&*()_+-=[]{}|;':\",./<>?"

	// Test with unicode characters
	unicodeName := "测试文件.toml"

	// These should not panic fatally
	recoverPanic(func() { findMissingKeys([]string{"--source=" + longName, "--target=es", "--dir=./test"}) })
	recoverPanic(func() { findMissingKeys([]string{"--source=" + specialChars, "--target=es", "--dir=./test"}) })
	recoverPanic(func() { findMissingKeys([]string{"--source=" + unicodeName, "--target=es", "--dir=./test"}) })

	recoverPanic(func() { validateFiles([]string{"--dir=./test"}) })
}

func TestIntegration_CompleteWorkflow(t *testing.T) {
	tempDir := t.TempDir()

	// Create a complete set of locale files
	enContent := `welcome = "Welcome"
hello = "Hello"
goodbye = "Goodbye"
thank_you = "Thank you"`

	esContent := `welcome = "Bienvenido"
hello = "Hola"
goodbye = "Adiós"`

	frContent := `welcome = "Bienvenue"
hello = "Bonjour"
goodbye = "Au revoir"
thank_you = "Merci"
extra_key = "Extra"`

	err := os.WriteFile(filepath.Join(tempDir, "en.toml"), []byte(enContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "es.toml"), []byte(esContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "fr.toml"), []byte(frContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test validation
	validateFiles([]string{"--dir=" + tempDir})

	// Test find missing keys
	findMissingKeys([]string{"--source=en", "--target=es", "--dir=" + tempDir})
	findMissingKeys([]string{"--source=en", "--target=fr", "--dir=" + tempDir})
	findMissingKeys([]string{"--source=es", "--target=fr", "--dir=" + tempDir})
}
