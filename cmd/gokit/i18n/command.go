package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/kdsmith18542/gokit/i18n"
)

// exitFunc is used for testability; defaults to os.Exit but can be overridden in tests
var exitFunc = os.Exit

// validatePath ensures the path is safe and within allowed directories
func validatePath(path string) error {
	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed: %s", path)
	}

	// Ensure path is valid
	_, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %s", path)
	}

	// Additional validation can be added here if needed
	return nil
}

// Run executes the i18n command-line tool.
// It parses subcommands and arguments to perform i18n-related operations
// like finding missing keys, validating files, or extracting keys.
func Run(args []string) {
	if len(args) < 1 {
		printI18nUsage()
		exitFunc(1)
		return
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "find-missing":
		findMissingKeys(subArgs)
	case "validate":
		validateFiles(subArgs)
	case "extract":
		extractKeys(subArgs)
	case "help":
		printI18nUsage()
	default:
		fmt.Printf("Unknown i18n subcommand: %s\n", subcommand)
		printI18nUsage()
		exitFunc(1)
		return
	}
}

func printI18nUsage() {
	fmt.Println("i18n - Manage internationalization files")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gokit i18n <subcommand> [options]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  find-missing  Find missing translation keys between locales")
	fmt.Println("  validate      Validate i18n files for syntax and completeness")
	fmt.Println("  extract       Extract translation keys from source code")
	fmt.Println("  help          Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gokit i18n find-missing --source=en --target=es --dir=./locales")
	fmt.Println("  gokit i18n validate --dir=./locales")
	fmt.Println("  gokit i18n extract --dir=./src --output=./locales --format=toml")
}

func findMissingKeys(args []string) {
	fs := flag.NewFlagSet("find-missing", flag.ExitOnError)
	source := fs.String("source", "", "Source locale (e.g., en)")
	target := fs.String("target", "", "Target locale (e.g., es)")
	dir := fs.String("dir", "./locales", "Directory containing locale files")

	if err := fs.Parse(args); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		fs.Usage()
		exitFunc(1)
		return
	}

	if *source == "" || *target == "" {
		fmt.Println("Error: --source and --target are required")
		fs.Usage()
		exitFunc(1)
		return
	}

	// Validate directory path
	if err := validatePath(*dir); err != nil {
		fmt.Printf("Error: %v\n", err)
		exitFunc(1)
		return
	}

	// Create a temporary manager to load the locales
	manager := i18n.NewManager(*dir)

	// Get available locales
	availableLocales := manager.GetAvailableLocales()

	// Check if source and target locales exist
	sourceExists := false
	targetExists := false
	for _, locale := range availableLocales {
		if locale == *source {
			sourceExists = true
		}
		if locale == *target {
			targetExists = true
		}
	}

	if !sourceExists {
		fmt.Printf("Error: Source locale '%s' not found in %s\n", *source, *dir)
		exitFunc(1)
		return
	}

	if !targetExists {
		fmt.Printf("Error: Target locale '%s' not found in %s\n", *target, *dir)
		exitFunc(1)
		return
	}

	// Get all keys from both locales by examining the locale files directly
	sourceKeys := getKeysFromLocaleFile(filepath.Join(*dir, *source+".toml"))
	targetKeys := getKeysFromLocaleFile(filepath.Join(*dir, *target+".toml"))

	// Find missing keys
	missingKeys := findMissingKeysInTarget(sourceKeys, targetKeys)
	extraKeys := findMissingKeysInTarget(targetKeys, sourceKeys)

	fmt.Printf("Comparing %s -> %s\n", *source, *target)
	fmt.Printf("Source file: %s\n", filepath.Join(*dir, *source+".toml"))
	fmt.Printf("Target file: %s\n", filepath.Join(*dir, *target+".toml"))
	fmt.Println()

	if len(missingKeys) == 0 && len(extraKeys) == 0 {
		fmt.Println("✅ All keys are synchronized between locales")
		return
	}

	if len(missingKeys) > 0 {
		fmt.Printf("❌ Missing keys in %s (%d):\n", *target, len(missingKeys))
		for _, key := range missingKeys {
			fmt.Printf("  - %s\n", key)
		}
		fmt.Println()
	}

	if len(extraKeys) > 0 {
		fmt.Printf("⚠️  Extra keys in %s (%d):\n", *target, len(extraKeys))
		for _, key := range extraKeys {
			fmt.Printf("  - %s\n", key)
		}
		fmt.Println()
	}
}

func validateFiles(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	dir := fs.String("dir", "./locales", "Directory containing locale files")

	if err := fs.Parse(args); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		fs.Usage()
		exitFunc(1)
		return
	}

	// Validate directory path
	if err := validatePath(*dir); err != nil {
		fmt.Printf("Error: %v\n", err)
		exitFunc(1)
		return
	}

	files, err := filepath.Glob(filepath.Join(*dir, "*.toml"))
	if err != nil {
		fmt.Printf("Error reading directory %s: %v\n", *dir, err)
		exitFunc(1)
		return
	}

	if len(files) == 0 {
		fmt.Printf("No .toml files found in %s\n", *dir)
		return
	}

	fmt.Printf("Validating %d locale files in %s\n\n", len(files), *dir)

	allValid := true

	for _, file := range files {
		locale := strings.TrimSuffix(filepath.Base(file), ".toml")
		fmt.Printf("Validating %s (%s)... ", file, locale)

		// Try to load the file to check if it's valid
		keys := getKeysFromLocaleFile(file)
		if keys == nil {
			fmt.Printf("❌ Error: Could not parse file\n")
			allValid = false
			continue
		}

		// Check for empty values
		emptyKeys := findEmptyKeysInFile(file)
		if len(emptyKeys) > 0 {
			fmt.Printf("⚠️  Warning: %d empty keys\n", len(emptyKeys))
			for _, key := range emptyKeys {
				fmt.Printf("    - %s\n", key)
			}
		} else {
			fmt.Println("✅ Valid")
		}
	}

	if allValid {
		fmt.Println("\n✅ All files are valid")
	} else {
		fmt.Println("\n❌ Some files have errors")
		exitFunc(1)
		return
	}
}

func extractKeys(args []string) {
	fs := flag.NewFlagSet("extract", flag.ExitOnError)
	dir := fs.String("dir", "./src", "Source directory to scan")
	output := fs.String("output", "./locales", "Output directory for locale files")
	format := fs.String("format", "toml", "Output format (toml, json)")

	if err := fs.Parse(args); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		fs.Usage()
		exitFunc(1)
		return
	}

	if *dir == "" || *output == "" {
		fmt.Println("Error: --dir and --output are required")
		fs.Usage()
		exitFunc(1)
		return
	}

	if *format != "toml" && *format != "json" {
		fmt.Println("Error: --format must be 'toml' or 'json'")
		fs.Usage()
		exitFunc(1)
		return
	}

	// Validate paths
	if err := validatePath(*dir); err != nil {
		fmt.Printf("Error with source directory: %v\n", err)
		exitFunc(1)
		return
	}
	if err := validatePath(*output); err != nil {
		fmt.Printf("Error with output directory: %v\n", err)
		exitFunc(1)
		return
	}

	fmt.Printf("Extracting translation keys from %s\n", *dir)
	fmt.Printf("Output format: %s\n", *format)
	fmt.Printf("Output directory: %s\n\n", *output)

	keys := scanSourceFiles(*dir)
	if len(keys) == 0 {
		fmt.Println("No translation keys found.")
		return
	}

	outputFile := filepath.Join(*output, "en."+*format) // Default to English for extracted keys
	err := writeKeysToFile(outputFile, keys, *format)
	if err != nil {
		fmt.Printf("Error writing keys to %s: %v\n", outputFile, err)
		exitFunc(1)
		return
	}

	fmt.Printf("Successfully extracted %d keys to %s\n", len(keys), outputFile)
}

// scanSourceFiles scans the given directory for .go files and extracts translation keys.
// It performs a simple regex match for translator.T("key"), t("key"), or i18n.Translate("key").
func scanSourceFiles(dir string) []string {
	var extractedKeys []string
	seenKeys := make(map[string]bool)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		// Validate file path before reading
		if err := validatePath(path); err != nil {
			fmt.Printf("Error with file path %s: %v\n", path, err)
			return nil // Continue scanning other files
		}

		content, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", path, err)
			return nil // Continue scanning other files
		}

		// Simple regex to find translation keys
		// Matches: translator.T("key"), t("key"), i18n.Translate("key")
		re := regexp.MustCompile(`(translator|t|i18n)\.(T|Translate)\("([^"]*)"\)`)
		matches := re.FindAllStringSubmatch(string(content), -1)

		for _, match := range matches {
			if len(match) > 3 {
				key := match[3]
				if !seenKeys[key] {
					extractedKeys = append(extractedKeys, key)
					seenKeys[key] = true
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error scanning source files in %s: %v\n", dir, err)
		exitFunc(1)
	}

	// Sort keys for consistent output
	sort.Strings(extractedKeys)
	return extractedKeys
}

// writeKeysToFile writes the extracted keys to a file in the specified format.
func writeKeysToFile(filepath string, keys []string, format string) error {
	m := make(map[string]string)
	for _, key := range keys {
		m[key] = "" // Initialize with empty string
	}

	var data []byte
	var err error

	switch format {
	case "toml":
		data, err = toml.Marshal(m)
	case "json":
		data, err = json.MarshalIndent(m, "", "  ")
	}

	if err != nil {
		return err
	}

	// Use more restrictive permissions (0600 instead of 0644)
	return ioutil.WriteFile(filepath, data, 0600)
}

func findMissingKeysInTarget(source, target []string) []string {
	targetMap := make(map[string]bool)
	for _, key := range target {
		targetMap[key] = true
	}

	var missing []string
	for _, key := range source {
		if !targetMap[key] {
			missing = append(missing, key)
		}
	}
	return missing
}

// getKeysFromLocaleFile parses a TOML file and returns all top-level keys.
func getKeysFromLocaleFile(filepath string) []string {
	// Validate file path before reading
	if err := validatePath(filepath); err != nil {
		return nil
	}

	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// findEmptyKeysInFile returns a list of keys with empty string values in a TOML file.
func findEmptyKeysInFile(filepath string) []string {
	// Validate file path before reading
	if err := validatePath(filepath); err != nil {
		return nil
	}

	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil
	}
	emptyKeys := []string{}
	for k, v := range m {
		if s, ok := v.(string); ok && s == "" {
			emptyKeys = append(emptyKeys, k)
		}
	}
	// Sort keys for consistent ordering across different runs
	sort.Strings(emptyKeys)
	return emptyKeys
}
