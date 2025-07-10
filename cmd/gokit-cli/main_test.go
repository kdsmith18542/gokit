package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(t *testing.T) {
	// Test that main doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("main() panicked: %v", r)
		}
	}()
}

func TestI18nCmd(t *testing.T) {
	cmd := I18nCmd()
	if cmd == nil {
		t.Fatal("i18nCmd() returned nil")
	}

	if cmd.Use != "i18n" {
		t.Errorf("Expected Use to be 'i18n', got '%s'", cmd.Use)
	}

	if cmd.Short != "Manage i18n message files" {
		t.Errorf("Expected Short to be 'Manage i18n message files', got '%s'", cmd.Short)
	}

	// Check that subcommands exist
	subcommands := cmd.Commands()
	if len(subcommands) != 2 {
		t.Errorf("Expected 2 subcommands, got %d", len(subcommands))
	}

	// Check find-missing subcommand
	findMissing := cmd.Commands()[0]
	if findMissing.Use != "find-missing" {
		t.Errorf("Expected first subcommand to be 'find-missing', got '%s'", findMissing.Use)
	}

	// Check lint subcommand
	lint := cmd.Commands()[1]
	if lint.Use != "lint" {
		t.Errorf("Expected second subcommand to be 'lint', got '%s'", lint.Use)
	}
}

func TestUploadCmd(t *testing.T) {
	cmd := UploadCmd()
	if cmd == nil {
		t.Fatal("uploadCmd() returned nil")
	}

	if cmd.Use != "upload" {
		t.Errorf("Expected Use to be 'upload', got '%s'", cmd.Use)
	}

	if cmd.Short != "Manage file upload backends" {
		t.Errorf("Expected Short to be 'Manage file upload backends', got '%s'", cmd.Short)
	}

	// Check that subcommands exist
	subcommands := cmd.Commands()
	if len(subcommands) != 5 {
		t.Errorf("Expected 5 subcommands, got %d", len(subcommands))
	}

	expectedSubcommands := []string{"delete-file [file]", "generate-url [file]", "list-files", "upload-file [file]", "verify-credentials"}
	for i, expected := range expectedSubcommands {
		if subcommands[i].Use != expected {
			t.Errorf("Expected subcommand %d to be '%s', got '%s'", i, expected, subcommands[i].Use)
		}
	}
}

func TestFindMissingKeys(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "test_locales")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Test with non-existent directory
	err = FindMissingKeys("/non/existent/directory")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}

	// Test with empty directory
	err = FindMissingKeys(tempDir)
	if err == nil {
		t.Error("Expected error for empty directory")
	}

	// Create test locale files
	enContent := `welcome = "Welcome"
hello = "Hello"
goodbye = "Goodbye"`

	esContent := `welcome = "Bienvenido"
hello = "Hola"`

	enFile := filepath.Join(tempDir, "en.toml")
	esFile := filepath.Join(tempDir, "es.toml")

	if err := os.WriteFile(enFile, []byte(enContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(esFile, []byte(esContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run findMissingKeys
	err = FindMissingKeys(tempDir)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Errorf("findMissingKeys failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Locale 'es' is missing 1 keys:") {
		t.Errorf("Expected output to show missing keys for Spanish, got: %s", output)
	}
	if !strings.Contains(output, "- goodbye") {
		t.Errorf("Expected output to show missing 'goodbye' key, got: %s", output)
	}
}

func TestLintLocaleFiles(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "test_locales")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Test with non-existent directory
	err = LintLocaleFiles("/non/existent/directory")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}

	// Test with empty directory
	err = LintLocaleFiles(tempDir)
	if err == nil {
		t.Error("Expected error for empty directory")
	}

	// Create valid test locale file
	validContent := `welcome = "Welcome"
hello = "Hello"`

	validFile := filepath.Join(tempDir, "en.toml")
	if err := os.WriteFile(validFile, []byte(validContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run lintLocaleFiles
	err = LintLocaleFiles(tempDir)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Errorf("lintLocaleFiles failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "✓ TOML syntax valid") {
		t.Errorf("Expected output to show valid TOML syntax, got: %s", output)
	}
	if !strings.Contains(output, "✓ All locale files passed linting!") {
		t.Errorf("Expected output to show all files passed linting, got: %s", output)
	}

	// Test with invalid TOML
	invalidContent := `welcome = "Welcome"
hello = "Hello"
invalid = "Missing quote`

	invalidFile := filepath.Join(tempDir, "invalid.toml")
	if err := os.WriteFile(invalidFile, []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Capture stdout again
	oldStdout = os.Stdout
	r, w, _ = os.Pipe()
	os.Stdout = w

	// Run lintLocaleFiles with invalid file
	err = LintLocaleFiles(tempDir)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	buf.Reset()
	io.Copy(&buf, r)

	if err == nil {
		t.Error("Expected error for invalid TOML file")
	}

	output = buf.String()
	if !strings.Contains(output, "❌ TOML syntax error:") {
		t.Errorf("Expected output to show TOML syntax error, got: %s", output)
	}
}

func TestLoadLocaleKeys(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "test_locale.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	// Test with non-existent file
	_, err = LoadLocaleKeys("/non/existent/file.toml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test with valid TOML content
	content := `welcome = "Welcome"
hello = "Hello"
nested.key = "Nested value"`

	if err := os.WriteFile(tempFile.Name(), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	keys, err := LoadLocaleKeys(tempFile.Name())
	if err != nil {
		t.Errorf("loadLocaleKeys failed: %v", err)
	}

	expectedKeys := []string{"welcome", "hello", "nested.key"}
	for _, expected := range expectedKeys {
		if !keys[expected] {
			t.Errorf("Expected key '%s' to be present", expected)
		}
	}

	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d keys, got %d", len(expectedKeys), len(keys))
	}
}

func TestCollectKeys(t *testing.T) {
	messages := map[string]interface{}{
		"welcome": "Welcome",
		"hello":   "Hello",
		"nested": map[string]interface{}{
			"key1": "Value1",
			"key2": "Value2",
		},
		"deep": map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": "Deep value",
			},
		},
	}

	keys := make(map[string]bool)
	CollectKeys(messages, "", keys)

	expectedKeys := []string{
		"welcome",
		"hello",
		"nested.key1",
		"nested.key2",
		"deep.level1.level2",
	}

	for _, expected := range expectedKeys {
		if !keys[expected] {
			t.Errorf("Expected key '%s' to be collected", expected)
		}
	}

	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d keys, got %d", len(expectedKeys), len(keys))
	}
}

func TestValidateTOMLSyntax(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "test.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	// Test with non-existent file
	err = ValidateTOMLSyntax("/non/existent/file.toml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test with valid TOML
	validContent := `welcome = "Welcome"
hello = "Hello"`

	if err := os.WriteFile(tempFile.Name(), []byte(validContent), 0644); err != nil {
		t.Fatal(err)
	}

	err = ValidateTOMLSyntax(tempFile.Name())
	if err != nil {
		t.Errorf("validateTOMLSyntax failed for valid TOML: %v", err)
	}

	// Test with invalid TOML
	invalidContent := `welcome = "Welcome"
hello = "Missing quote`

	if err := os.WriteFile(tempFile.Name(), []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}

	err = ValidateTOMLSyntax(tempFile.Name())
	if err == nil {
		t.Error("Expected error for invalid TOML")
	}
}

func TestCheckCommonIssues(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "test.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	// Test with non-existent file
	issues := CheckCommonIssues("/non/existent/file.toml")
	if len(issues) == 0 {
		t.Error("Expected issues for non-existent file")
	}

	// Test with valid content (no issues)
	validContent := `welcome = "Welcome"
hello = "Hello"`

	if err := os.WriteFile(tempFile.Name(), []byte(validContent), 0644); err != nil {
		t.Fatal(err)
	}

	issues = CheckCommonIssues(tempFile.Name())
	if len(issues) != 0 {
		t.Errorf("Expected no issues for valid content, got: %v", issues)
	}

	// Test with unquoted values
	unquotedContent := `welcome = Welcome
hello = "Hello"`

	if err := os.WriteFile(tempFile.Name(), []byte(unquotedContent), 0644); err != nil {
		t.Fatal(err)
	}

	issues = CheckCommonIssues(tempFile.Name())
	if len(issues) == 0 {
		t.Error("Expected issues for unquoted values")
	}

	// Test with unquoted values (this should be detected)
	unquotedContent2 := `welcome = "Welcome"
hello = Hello`

	if err := os.WriteFile(tempFile.Name(), []byte(unquotedContent2), 0644); err != nil {
		t.Fatal(err)
	}

	issues = CheckCommonIssues(tempFile.Name())
	if len(issues) == 0 {
		t.Error("Expected issues for unquoted values")
	}
}

func TestGetSortedKeys(t *testing.T) {
	m := map[string]map[string]bool{
		"zebra": {"key1": true, "key2": true},
		"alpha": {"key3": true},
		"beta":  {"key4": true, "key5": true},
	}

	sorted := GetSortedKeys(m)

	expected := []string{"alpha", "beta", "zebra"}
	if len(sorted) != len(expected) {
		t.Errorf("Expected %d keys, got %d", len(expected), len(sorted))
	}

	for i, expectedKey := range expected {
		if sorted[i] != expectedKey {
			t.Errorf("Expected key at position %d to be '%s', got '%s'", i, expectedKey, sorted[i])
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1500, "1.5 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TB"},
	}

	for _, test := range tests {
		result := FormatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", test.bytes, result, test.expected)
		}
	}
}

func TestCommandExecution(t *testing.T) {
	// Test i18n find-missing command execution
	cmd := I18nCmd()
	findMissingCmd := cmd.Commands()[0]

	// Create temporary directory with test files
	tempDir, err := os.MkdirTemp("", "test_locales")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	enContent := `welcome = "Welcome"
hello = "Hello"`

	esContent := `welcome = "Bienvenido"`

	enFile := filepath.Join(tempDir, "en.toml")
	esFile := filepath.Join(tempDir, "es.toml")

	if err := os.WriteFile(enFile, []byte(enContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(esFile, []byte(esContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Set the dir flag on the parent command
	err = cmd.PersistentFlags().Set("dir", tempDir)
	if err != nil {
		t.Fatalf("Failed to set dir flag: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute the command
	findMissingCmd.Run(findMissingCmd, []string{})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	output := buf.String()
	if !strings.Contains(output, "Locale 'es' is missing 1 keys:") {
		t.Errorf("Expected output to show missing keys, got: %s", output)
	}
	if !strings.Contains(output, "- hello") {
		t.Errorf("Expected output to show missing 'hello' key, got: %s", output)
	}
}

func TestUploadCommandExecution(t *testing.T) {
	cmd := UploadCmd()

	// Test verify-credentials command with invalid backend (it's the last command)
	verifyCmd := cmd.Commands()[4]

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute with invalid backend
	cmd.PersistentFlags().Set("backend", "invalid")
	verifyCmd.Run(verifyCmd, []string{})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	output := buf.String()
	if !strings.Contains(output, "Supported backends: s3, gcs, azure") {
		t.Errorf("Expected output to show supported backends, got: %s", output)
	}
}
