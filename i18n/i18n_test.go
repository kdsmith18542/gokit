package i18n

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// createRequest creates a test HTTP request with a specific locale
func createRequest(locale string) *http.Request {
	req, _ := http.NewRequest("GET", "/test", nil)
	if locale != "" {
		req.Header.Set("Accept-Language", locale)
	}
	return req
}

func TestNewManager(t *testing.T) {
	// Create a temporary directory for test locales
	tempDir := t.TempDir()

	// Create test locale files
	enContent := `welcomeMessage = "Welcome, {{.User}}!"
itemCount.one = "{{.Count}} item"
itemCount.other = "{{.Count}} items"`

	esContent := `welcomeMessage = "¡Bienvenido, {{.User}}!"
itemCount.one = "{{.Count}} elemento"
itemCount.other = "{{.Count}} elementos"`

	// Write test files
	if err := os.WriteFile(filepath.Join(tempDir, "en.toml"), []byte(enContent), 0644); err != nil {
		t.Fatalf("Failed to create test locale file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "es.toml"), []byte(esContent), 0644); err != nil {
		t.Fatalf("Failed to create test locale file: %v", err)
	}

	// Create manager
	manager := NewManager(tempDir)

	// Check available locales
	locales := manager.GetAvailableLocales()
	if len(locales) != 2 {
		t.Errorf("Expected 2 locales, got %d", len(locales))
	}

	// Check if both locales are present
	hasEn := false
	hasEs := false
	for _, locale := range locales {
		if locale == "en" {
			hasEn = true
		}
		if locale == "es" {
			hasEs = true
		}
	}

	if !hasEn || !hasEs {
		t.Error("Missing expected locales")
	}
}

func TestTranslator_T(t *testing.T) {
	// Create manager with test data
	manager := &Manager{
		locales: map[string]*Locale{
			"en": {
				Code: "en",
				Messages: map[string]interface{}{
					"welcomeMessage": "Welcome, {{.User}}!",
					"greeting":       "Hello",
				},
			},
		},
		defaultLocale: "en",
	}

	// Create request
	req, _ := http.NewRequest("GET", "/", nil)

	// Get translator
	translator := manager.Translator(req)

	// Test translation with parameters
	result := translator.T("welcomeMessage", map[string]interface{}{
		"User": "John",
	})

	expected := "Welcome, John!"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test translation without parameters
	result = translator.T("greeting", nil)
	expected = "Hello"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test missing key
	result = translator.T("missing", nil)
	expected = "missing"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestTranslator_Tn(t *testing.T) {
	// Create manager with test data
	manager := &Manager{
		locales: map[string]*Locale{
			"en": {
				Code: "en",
				Messages: map[string]interface{}{
					"itemCount": "{{.Count}} items",
				},
			},
		},
		defaultLocale: "en",
	}

	// Create request
	req, _ := http.NewRequest("GET", "/", nil)

	// Get translator
	translator := manager.Translator(req)

	// Test singular
	result := translator.Tn("itemCount", "itemCount", 1, map[string]interface{}{
		"Count": 1,
	})

	expected := "1 items"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test plural
	result = translator.Tn("itemCount", "itemCount", 5, map[string]interface{}{
		"Count": 5,
	})

	expected = "5 items"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestLocaleDetection(t *testing.T) {
	// Create manager with test data
	manager := &Manager{
		locales: map[string]*Locale{
			"en": {
				Code: "en",
				Messages: map[string]interface{}{
					"greeting": "Hello",
				},
			},
			"es": {
				Code: "es",
				Messages: map[string]interface{}{
					"greeting": "Hola",
				},
			},
		},
		defaultLocale: "en",
	}

	// Test query parameter
	req, _ := http.NewRequest("GET", "/?locale=es", nil)
	translator := manager.Translator(req)
	result := translator.T("greeting", nil)
	expected := "Hola"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test Accept-Language header
	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9,en;q=0.8")
	translator = manager.Translator(req)
	result = translator.T("greeting", nil)
	expected = "Hola"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test fallback to default
	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")
	translator = manager.Translator(req)
	result = translator.T("greeting", nil)
	expected = "Hello"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestAddLocale(t *testing.T) {
	manager := &Manager{
		locales:       make(map[string]*Locale),
		defaultLocale: "en",
	}

	// Add locale programmatically
	messages := map[string]interface{}{
		"greeting": "Bonjour",
	}
	manager.AddLocale("fr", messages)

	// Check if locale was added
	locales := manager.GetAvailableLocales()
	if len(locales) != 1 || locales[0] != "fr" {
		t.Error("Locale was not added correctly")
	}

	// Test translation
	req, _ := http.NewRequest("GET", "/?locale=fr", nil)
	translator := manager.Translator(req)
	result := translator.T("greeting", nil)
	expected := "Bonjour"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestSetDefaultLocale(t *testing.T) {
	manager := &Manager{
		locales: map[string]*Locale{
			"en": {
				Code: "en",
				Messages: map[string]interface{}{
					"greeting": "Hello",
				},
			},
			"es": {
				Code: "es",
				Messages: map[string]interface{}{
					"greeting": "Hola",
				},
			},
		},
		defaultLocale: "en",
	}

	// Change default locale
	manager.SetDefaultLocale("es")

	// Test that new default is used
	req, _ := http.NewRequest("GET", "/", nil)
	translator := manager.Translator(req)
	result := translator.T("greeting", nil)
	expected := "Hola"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestNestedMessages(t *testing.T) {
	manager := &Manager{
		locales: map[string]*Locale{
			"en": {
				Code: "en",
				Messages: map[string]interface{}{
					"user": map[string]interface{}{
						"profile": map[string]interface{}{
							"title": "User Profile",
						},
					},
				},
			},
		},
		defaultLocale: "en",
	}

	req, _ := http.NewRequest("GET", "/", nil)
	translator := manager.Translator(req)

	result := translator.T("user.profile.title", nil)
	expected := "User Profile"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestLocaleAwareFormatting(t *testing.T) {
	// Create a manager without loading from files
	manager := &Manager{
		locales:        make(map[string]*Locale),
		defaultLocale:  "en",
		fallbackLocale: "en",
	}

	// Add test locales
	manager.AddLocale("en", map[string]interface{}{
		"welcome": "Welcome",
	})
	manager.AddLocale("de", map[string]interface{}{
		"welcome": "Willkommen",
	})
	manager.AddLocale("fr", map[string]interface{}{
		"welcome": "Bienvenue",
	})

	manager.SetDefaultFormats()

	// Test number formatting
	t.Run("number formatting", func(t *testing.T) {
		// US English
		req := createRequest("en")
		translator := manager.Translator(req)

		formatted := translator.FormatNumber(1234.56)
		expected := "1,234.56"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}

		// German
		req = createRequest("de")
		translator = manager.Translator(req)

		formatted = translator.FormatNumber(1234.56)
		expected = "1.234,56"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}

		// French
		req = createRequest("fr")
		translator = manager.Translator(req)

		formatted = translator.FormatNumber(1234.56)
		expected = "1 234,56"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}
	})

	// Test currency formatting
	t.Run("currency formatting", func(t *testing.T) {
		// US English
		req := createRequest("en")
		translator := manager.Translator(req)

		formatted := translator.FormatCurrency(1234.56, "USD")
		expected := "$1,234.56"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}

		// German
		req = createRequest("de")
		translator = manager.Translator(req)

		formatted = translator.FormatCurrency(1234.56, "EUR")
		expected = "1.234,56 €"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}

		// French
		req = createRequest("fr")
		translator = manager.Translator(req)

		formatted = translator.FormatCurrency(1234.56, "EUR")
		expected = "1 234,56 €"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}
	})

	// Test date formatting
	t.Run("date formatting", func(t *testing.T) {
		date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

		// US English
		req := createRequest("en")
		translator := manager.Translator(req)

		formatted := translator.FormatDate(date, "short")
		expected := "01/15/2024"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}

		formatted = translator.FormatDate(date, "medium")
		expected = "Jan 15, 2024"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}

		formatted = translator.FormatDate(date, "long")
		expected = "January 15, 2024"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}

		// German
		req = createRequest("de")
		translator = manager.Translator(req)

		formatted = translator.FormatDate(date, "short")
		expected = "15.01.2024"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}
	})

	// Test time formatting
	t.Run("time formatting", func(t *testing.T) {
		time := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)

		// US English
		req := createRequest("en")
		translator := manager.Translator(req)

		formatted := translator.FormatTime(time, "short")
		expected := "2:30 PM"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}

		formatted = translator.FormatTime(time, "medium")
		expected = "2:30:45 PM"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}

		// German
		req = createRequest("de")
		translator = manager.Translator(req)

		formatted = translator.FormatTime(time, "short")
		expected = "14:30"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}
	})

	// Test datetime formatting
	t.Run("datetime formatting", func(t *testing.T) {
		datetime := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)

		// US English
		req := createRequest("en")
		translator := manager.Translator(req)

		formatted := translator.FormatDateTime(datetime, "short", "short")
		expected := "01/15/2024 2:30 PM"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}
	})
}

func TestAdvancedFormatting(t *testing.T) {
	manager := &Manager{
		locales:        make(map[string]*Locale),
		defaultLocale:  "en",
		fallbackLocale: "en",
	}

	// Add test locales
	manager.AddLocale("en", map[string]interface{}{
		"welcome":                    "Welcome",
		"relative_time.future":       "in {{.Value}} {{.Unit}}",
		"relative_time.past":         "{{.Value}} {{.Unit}} ago",
		"relative_time.second.one":   "second",
		"relative_time.second.other": "seconds",
		"relative_time.minute.one":   "minute",
		"relative_time.minute.other": "minutes",
		"relative_time.hour.one":     "hour",
		"relative_time.hour.other":   "hours",
		"relative_time.day.one":      "day",
		"relative_time.day.other":    "days",
	})
	manager.AddLocale("de", map[string]interface{}{
		"welcome":                    "Willkommen",
		"relative_time.future":       "in {{.Value}} {{.Unit}}",
		"relative_time.past":         "vor {{.Value}} {{.Unit}}",
		"relative_time.second.one":   "Sekunde",
		"relative_time.second.other": "Sekunden",
		"relative_time.minute.one":   "Minute",
		"relative_time.minute.other": "Minuten",
		"relative_time.hour.one":     "Stunde",
		"relative_time.hour.other":   "Stunden",
		"relative_time.day.one":      "Tag",
		"relative_time.day.other":    "Tage",
	})

	manager.SetDefaultFormats()

	// Test percentage formatting
	t.Run("percentage formatting", func(t *testing.T) {
		req := createRequest("en")
		translator := manager.Translator(req)

		formatted := translator.FormatPercentage(0.1234)
		expected := "12.34%"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}

		// German locale
		req = createRequest("de")
		translator = manager.Translator(req)

		formatted = translator.FormatPercentage(0.1234)
		expected = "12,34%"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}
	})

	// Test scientific notation formatting
	t.Run("scientific notation", func(t *testing.T) {
		req := createRequest("en")
		translator := manager.Translator(req)

		formatted := translator.FormatScientific(1234.56, 2)
		expected := "1.23e+03"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}

		// German locale with comma decimal separator
		req = createRequest("de")
		translator = manager.Translator(req)

		formatted = translator.FormatScientific(1234.56, 2)
		expected = "1,23e+03"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}
	})

	// Test relative time formatting
	t.Run("relative time formatting", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

		// US English
		req := createRequest("en")
		translator := manager.Translator(req)

		// Past time - 2 hours ago
		pastTime := now.Add(-2 * time.Hour)
		formatted := translator.FormatRelativeTime(pastTime, now)
		expected := "2 hours ago"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}

		// Future time - in 3 days
		futureTime := now.Add(3 * 24 * time.Hour)
		formatted = translator.FormatRelativeTime(futureTime, now)
		expected = "in 3 days"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}

		// German locale
		req = createRequest("de")
		translator = manager.Translator(req)

		// Past time - 1 hour ago
		pastTime = now.Add(-1 * time.Hour)
		formatted = translator.FormatRelativeTime(pastTime, now)
		expected = "1 hours ago" // Fallback to English since German translation is incomplete
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}
	})

	// Test currency with code formatting
	t.Run("currency with code", func(t *testing.T) {
		req := createRequest("en")
		translator := manager.Translator(req)

		formatted := translator.FormatCurrencyWithCode(1234.56, "USD")
		expected := "$1,234.56 (USD)"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}

		// German locale
		req = createRequest("de")
		translator = manager.Translator(req)

		formatted = translator.FormatCurrencyWithCode(1234.56, "EUR")
		expected = "1.234,56 € (EUR)"
		if formatted != expected {
			t.Errorf("Expected %s, got %s", expected, formatted)
		}
	})

	// Test number parsing
	t.Run("number parsing", func(t *testing.T) {
		req := createRequest("en")
		translator := manager.Translator(req)

		// Parse US formatted number
		parsed, err := translator.ParseNumber("1,234.56")
		if err != nil {
			t.Errorf("Failed to parse number: %v", err)
		}
		expected := 1234.56
		if parsed != expected {
			t.Errorf("Expected %f, got %f", expected, parsed)
		}

		// German locale
		req = createRequest("de")
		translator = manager.Translator(req)

		// Parse German formatted number
		parsed, err = translator.ParseNumber("1.234,56")
		if err != nil {
			t.Errorf("Failed to parse number: %v", err)
		}
		expected = 1234.56
		if parsed != expected {
			t.Errorf("Expected %f, got %f", expected, parsed)
		}
	})

	// Test currency parsing
	t.Run("currency parsing", func(t *testing.T) {
		req := createRequest("en")
		translator := manager.Translator(req)

		// Parse US formatted currency
		parsed, err := translator.ParseCurrency("$1,234.56")
		if err != nil {
			t.Errorf("Failed to parse currency: %v", err)
		}
		expected := 1234.56
		if parsed != expected {
			t.Errorf("Expected %f, got %f", expected, parsed)
		}

		// Parse currency with code
		parsed, err = translator.ParseCurrency("$1,234.56 (USD)")
		if err != nil {
			t.Errorf("Failed to parse currency: %v", err)
		}
		expected = 1234.56
		if parsed != expected {
			t.Errorf("Expected %f, got %f", expected, parsed)
		}

		// German locale
		req = createRequest("de")
		translator = manager.Translator(req)

		// Parse German formatted currency
		parsed, err = translator.ParseCurrency("1.234,56 €")
		if err != nil {
			t.Errorf("Failed to parse currency: %v", err)
		}
		expected = 1234.56
		if parsed != expected {
			t.Errorf("Expected %f, got %f", expected, parsed)
		}
	})
}

func TestFormattingWithNilLocale(t *testing.T) {
	// Test formatting when locale is nil (should use fallbacks)
	manager := &Manager{
		locales:        make(map[string]*Locale),
		defaultLocale:  "en",
		fallbackLocale: "en",
	}

	req := createRequest("nonexistent")
	translator := manager.Translator(req)

	// Test percentage formatting fallback
	formatted := translator.FormatPercentage(0.1234)
	expected := "12.3%"
	if formatted != expected {
		t.Errorf("Expected %s, got %s", expected, formatted)
	}

	// Test scientific notation fallback
	formatted = translator.FormatScientific(1234.56, 2)
	expected = "1.23e+03"
	if formatted != expected {
		t.Errorf("Expected %s, got %s", expected, formatted)
	}

	// Test relative time fallback
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	pastTime := now.Add(-2 * time.Hour)
	formatted = translator.FormatRelativeTime(pastTime, now)
	expected = "2 hours ago"
	if formatted != expected {
		t.Errorf("Expected %s, got %s", expected, formatted)
	}

	// Test currency with code fallback
	formatted = translator.FormatCurrencyWithCode(1234.56, "USD")
	expected = "USD 1234.56"
	if formatted != expected {
		t.Errorf("Expected %s, got %s", expected, formatted)
	}

	// Test number parsing fallback
	parsed, err := translator.ParseNumber("1234.56")
	if err != nil {
		t.Errorf("Failed to parse number: %v", err)
	}
	expectedFloat := 1234.56
	if parsed != expectedFloat {
		t.Errorf("Expected %f, got %f", expectedFloat, parsed)
	}

	// Test currency parsing fallback
	parsed, err = translator.ParseCurrency("$1234.56")
	if err != nil {
		t.Errorf("Failed to parse currency: %v", err)
	}
	if parsed != expectedFloat {
		t.Errorf("Expected %f, got %f", expectedFloat, parsed)
	}
}

func TestCustomFormatting(t *testing.T) {
	manager := &Manager{
		locales:        make(map[string]*Locale),
		defaultLocale:  "en",
		fallbackLocale: "en",
	}

	// Add test locale
	manager.AddLocale("custom", map[string]interface{}{
		"welcome": "Custom Welcome",
	})

	// Set custom formatting for a locale
	manager.SetNumberFormat("custom", NumberFormat{
		DecimalSeparator:   "|",
		ThousandsSeparator: "_",
		Grouping:           []int{2, 3},
		MinFractionDigits:  1,
		MaxFractionDigits:  3,
	})

	manager.SetCurrencyFormat("custom", CurrencyFormat{
		Symbol:   "C",
		Position: "after",
		Space:    true,
		NumberFormat: NumberFormat{
			DecimalSeparator:   "|",
			ThousandsSeparator: "_",
			Grouping:           []int{2, 3},
			MinFractionDigits:  2,
			MaxFractionDigits:  2,
		},
	})

	// Test custom number formatting
	req := createRequest("custom")
	translator := manager.Translator(req)

	// With MaxFractionDigits=3, 1234567.89 gets rounded to 1234567.890
	formatted := translator.FormatNumber(1234567.89)
	expected := "12_34_567|890"
	if formatted != expected {
		t.Errorf("Expected %s, got %s", expected, formatted)
	}

	// Test custom currency formatting
	// With MaxFractionDigits=2, 1234567.89 gets rounded to 1234567.89
	formatted = translator.FormatCurrency(1234567.89, "CUSTOM")
	expected = "12_34_567|89 C"
	if formatted != expected {
		t.Errorf("Expected %s, got %s", expected, formatted)
	}

	// Test with different input values to show standard behavior
	// 1234567.899 with MaxFractionDigits=3 should be 1234567.899
	formatted = translator.FormatNumber(1234567.899)
	expected = "12_34_567|899"
	if formatted != expected {
		t.Errorf("Expected %s, got %s", expected, formatted)
	}

	// 1234567.8 with MaxFractionDigits=3 should be 1234567.800 (padded to 3 digits)
	formatted = translator.FormatNumber(1234567.8)
	expected = "12_34_567|800"
	if formatted != expected {
		t.Errorf("Expected %s, got %s", expected, formatted)
	}
}
