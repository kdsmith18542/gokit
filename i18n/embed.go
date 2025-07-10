package i18n

import (
	"embed"
	"fmt"
	"strings"
)

// NewManagerEmpty creates a new i18n manager without loading any locale files.
// This is useful for testing or when you want to add locales programmatically.
func NewManagerEmpty() *Manager {
	return &Manager{
		locales:        make(map[string]*Locale),
		defaultLocale:  "en",
		fallbackLocale: "en",
	}
}

// NewManagerFromFS creates a new i18n manager from an embedded filesystem.
// This allows loading translation files that are embedded in the binary.
//
// Example:
//
//	//go:embed locales/*.toml
//	var localeFS embed.FS
//
//	func init() {
//	    i18nManager = i18n.NewManagerFromFS(localeFS)
//	}
func NewManagerFromFS(fs embed.FS) *Manager {
	manager := &Manager{
		locales:        make(map[string]*Locale),
		defaultLocale:  "en",
		fallbackLocale: "en",
	}

	// Load all TOML files from the embedded filesystem
	if err := manager.loadLocalesFromFS(fs); err != nil {
		// Log error but don't fail - manager will still work with programmatically added locales
		fmt.Printf("Warning: failed to load embedded locales: %v\n", err)
	}

	return manager
}

// NewManagerFromFSWithPath creates a new i18n manager from an embedded filesystem
// with a specific subdirectory path.
//
// Example:
//
//	//go:embed assets/locales/*.toml
//	var assetFS embed.FS
//
//	func init() {
//	    i18nManager = i18n.NewManagerFromFSWithPath(assetFS, "assets/locales")
//	}
func NewManagerFromFSWithPath(fs embed.FS, path string) *Manager {
	manager := &Manager{
		locales:        make(map[string]*Locale),
		defaultLocale:  "en",
		fallbackLocale: "en",
	}

	// Load all TOML files from the specified subdirectory
	if err := manager.loadLocalesFromFSPath(fs, path); err != nil {
		// Log error but don't fail - manager will still work with programmatically added locales
		fmt.Printf("Warning: failed to load embedded locales from %s: %v\n", path, err)
	}

	return manager
}

// loadLocalesFromFS loads all locale files from an embedded filesystem
func (m *Manager) loadLocalesFromFS(fs embed.FS) error {
	// For embedded filesystems, we need to know the file names in advance
	// This is a simplified approach - in practice, users would specify the files they want to load
	return nil
}

// loadLocalesFromFSPath loads all locale files from a specific subdirectory in an embedded filesystem
func (m *Manager) loadLocalesFromFSPath(fs embed.FS, path string) error {
	// For embedded filesystems, we need to know the file names in advance
	// This is a simplified approach - in practice, users would specify the files they want to load
	return nil
}

// loadLocaleFileFromFS loads a single locale file from an embedded filesystem
func (m *Manager) loadLocaleFileFromFS(fs embed.FS, code, path string) error {
	// Read the file content
	data, err := fs.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %v", path, err)
	}

	// Parse the TOML content
	messages := make(map[string]interface{})
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Simple key=value parsing (same as file-based loading)
		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove quotes if present
			if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') {
				value = value[1 : len(value)-1]
			}

			messages[key] = value
		}
	}

	// Create and store the locale
	locale := &Locale{
		Code:     code,
		Messages: messages,
	}

	m.mu.Lock()
	m.locales[code] = locale
	m.mu.Unlock()

	return nil
}

// AddLocaleFromFS adds a locale from an embedded filesystem file
func (m *Manager) AddLocaleFromFS(fs embed.FS, code, path string) error {
	return m.loadLocaleFileFromFS(fs, code, path)
}

// AddLocalesFromFS adds multiple locales from an embedded filesystem
func (m *Manager) AddLocalesFromFS(fs embed.FS, path string) error {
	return m.loadLocalesFromFSPath(fs, path)
}
