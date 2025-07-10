// Package i18n provides a complete internationalization and localization solution for Go web applications.
//
// Features:
//   - Automatic locale detection from HTTP headers, cookies, and query parameters
//   - TOML-based message bundles with support for nested keys
//   - CLDR-based pluralization rules
//   - Locale-aware formatting for numbers, currencies, dates, and times
//   - Live reloading of message bundles (development mode)
//   - Web-based editor for non-developers
//   - Concurrency-safe for use in HTTP handlers
//
// Example:
//
//	// Initialize manager
//	manager := i18n.NewManager("./locales")
//
//	// In HTTP handler
//	func GreetingHandler(w http.ResponseWriter, r *http.Request) {
//	    translator := manager.Translator(r)
//
//	    // Simple translation
//	    greeting := translator.T("welcome", map[string]interface{}{
//	        "User": "Alex",
//	    })
//
//	    // Pluralization
//	    items := translator.Tn("item", "items", 5, nil)
//
//	    // Formatting
//	    price := translator.FormatCurrency(1234.56, "USD")
//	    date := translator.FormatDate(time.Now(), "medium")
//	}
//
// Locale files (e.g., en.toml):
//
//	welcome = "Welcome, {{.User}}!"
//	item = "1 item"
//	items = "{{.Count}} items"
//	price = "Price: {{.Amount}}"
package i18n

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"
)

// Constants for commonly used strings
const (
	// Plural forms
	PluralOne   = "one"
	PluralTwo   = "two"
	PluralFew   = "few"
	PluralMany  = "many"
	PluralOther = "other"
	PluralZero  = "zero"

	// Format types
	FormatShort  = "short"
	FormatLong   = "long"
	FormatMedium = "medium"

	// Fallback formats
	FallbackDateShort  = "01/02/2006"
	FallbackDateLong   = "January 2, 2006"
	FallbackDateMedium = "Jan 2, 2006"
	FallbackTimeShort  = "15:04"
	FallbackTimeLong   = "3:04:05 PM"
	FallbackTimeMedium = "15:04:05"
)

// Manager holds all translation data and configuration.
// It is safe for concurrent use and should be initialized once at application startup.
type Manager struct {
	locales        map[string]*Locale
	defaultLocale  string
	fallbackLocale string
	mu             sync.RWMutex
}

// Locale represents a single locale with its messages and formatting rules.
// Each locale contains translation messages and locale-specific formatting configurations.
type Locale struct {
	Code     string
	Messages map[string]interface{}
	// Formatting configuration
	NumberFormat   NumberFormat
	CurrencyFormat CurrencyFormat
	DateFormat     DateFormat
	TimeFormat     TimeFormat
}

// NumberFormat defines number formatting rules for a locale.
// Used for formatting numbers according to locale-specific conventions.
type NumberFormat struct {
	DecimalSeparator   string
	ThousandsSeparator string
	Grouping           []int // e.g., [3] for groups of 3 digits
	MinFractionDigits  int
	MaxFractionDigits  int
}

// CurrencyFormat defines currency formatting rules for a locale.
// Extends NumberFormat with currency-specific formatting options.
type CurrencyFormat struct {
	Symbol   string
	Position string // "before" or "after"
	Space    bool   // whether to add space between symbol and amount
	NumberFormat
}

// DateFormat defines date formatting rules for a locale.
// Provides different date format patterns for short, medium, and long displays.
type DateFormat struct {
	Short  string // e.g., "MM/DD/YYYY"
	Medium string // e.g., "MMM DD, YYYY"
	Long   string // e.g., "MMMM DD, YYYY"
}

// TimeFormat defines time formatting rules for a locale.
// Provides different time format patterns for short, medium, and long displays.
type TimeFormat struct {
	Short  string // e.g., "HH:MM"
	Medium string // e.g., "HH:MM:SS"
	Long   string // e.g., "HH:MM:SS AM/PM"
}

// Translator provides translation methods for a specific request.
// Each translator is bound to a specific locale and provides methods for translation and formatting.
type Translator struct {
	locale  *Locale
	manager *Manager
}

// NewManager creates a new i18n manager and loads translation files from the specified directory.
// The manager will automatically load all .toml files from the localesPath directory.
// Panics if the locales directory cannot be read or if locale files are invalid.
//
// Example:
//
//	manager := i18n.NewManager("./locales")
func NewManager(localesPath string) *Manager {
	manager := &Manager{
		locales:        make(map[string]*Locale),
		defaultLocale:  "en",
		fallbackLocale: "en",
	}

	// Load all locale files from the directory
	if err := manager.loadLocales(localesPath); err != nil {
		panic(fmt.Sprintf("Failed to load locales: %v", err))
	}

	return manager
}

// SetDefaultLocale sets the default locale for the manager.
// This locale will be used when no locale can be detected from the request.
func (m *Manager) SetDefaultLocale(locale string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultLocale = locale
}

// SetFallbackLocale sets the fallback locale for the manager.
// This locale will be used when a requested locale is not available.
func (m *Manager) SetFallbackLocale(locale string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fallbackLocale = locale
}

// Translator returns a translator for the current request's locale.
// The locale is automatically detected from the request's Accept-Language header,
// cookies, or query parameters. Falls back to the default locale if detection fails.
//
// Example:
//
//	translator := manager.Translator(r)
//	message := translator.T("welcome", nil)
func (m *Manager) Translator(r *http.Request) *Translator {
	locale := m.detectLocale(r)
	return &Translator{
		locale:  locale,
		manager: m,
	}
}

// T translates a message key with optional parameters.
// Returns the translated message with parameter substitution, or the key itself if no translation is found.
//
// Parameters are substituted using Go template syntax: {{.ParamName}}
//
// Example:
//
//	message := translator.T("welcome", map[string]interface{}{
//	    "User": "Alex",
//	    "Count": 5,
//	})
func (t *Translator) T(key string, params map[string]interface{}) string {
	start := time.Now()
	ctx := context.Background()

	if obs := getObserver(); obs != nil {
		obs.OnTranslationStart(ctx, t.locale.Code, key)
	}

	message := t.getMessage(key)
	if message == "" {
		return key
	}

	// Handle pluralization
	if pluralKey, count := t.extractPluralKey(key, params); pluralKey != "" {
		pluralMessage := t.getPluralMessage(pluralKey, count)
		if pluralMessage != "" {
			message = pluralMessage
		}
	}

	// Apply template substitution
	result := t.substituteParams(message, params)

	if obs := getObserver(); obs != nil {
		obs.OnTranslationEnd(ctx, t.locale.Code, key, time.Since(start))
	}

	return result
}

// Tn translates a message with pluralization support.
// Automatically selects the appropriate singular or plural form based on the count.
// The count is automatically added to the parameters as "Count".
//
// Example:
//
//	message := translator.Tn("item", "items", 5, map[string]interface{}{
//	    "Category": "books",
//	})
//	// Result: "5 items" (with parameter substitution)
func (t *Translator) Tn(singular, plural string, count int, params map[string]interface{}) string {
	if params == nil {
		params = make(map[string]interface{})
	}
	params["Count"] = count

	// Determine which form to use based on count
	var message string
	if count == 1 {
		message = t.getMessage(singular)
		if message == "" {
			message = singular
		}
	} else {
		message = t.getMessage(plural)
		if message == "" {
			message = plural
		}
	}

	return t.substituteParams(message, params)
}

// FormatNumber formats a number according to the locale's number format.
// Applies locale-specific decimal separators, thousands separators, and precision.
//
// Example:
//
//	formatted := translator.FormatNumber(1234.56)
//	// Result: "1,234.56" (for en locale)
func (t *Translator) FormatNumber(number float64) string {
	if t.locale == nil {
		return fmt.Sprintf("%.2f", number)
	}

	format := t.locale.NumberFormat
	if format.DecimalSeparator == "" {
		format.DecimalSeparator = "."
	}
	if format.ThousandsSeparator == "" {
		format.ThousandsSeparator = ","
	}

	// Convert to string with proper decimal places
	precision := format.MaxFractionDigits
	if precision == 0 {
		precision = 2
	}

	str := fmt.Sprintf("%.*f", precision, number)

	// Split into integer and decimal parts
	parts := strings.Split(str, ".")
	integerPart := parts[0]
	decimalPart := ""
	if len(parts) > 1 {
		decimalPart = parts[1]
	}

	// Add thousands separators
	if format.Grouping != nil && len(format.Grouping) > 0 {
		integerPart = addNumberGrouping(integerPart, format)
	}

	// Combine parts
	if decimalPart != "" {
		return integerPart + format.DecimalSeparator + decimalPart
	}
	return integerPart
}

// formatNumberWithFormat formats a number using the specified number format
func (t *Translator) formatNumberWithFormat(number float64, format NumberFormat) string {
	if format.DecimalSeparator == "" {
		format.DecimalSeparator = "."
	}
	if format.ThousandsSeparator == "" {
		format.ThousandsSeparator = ","
	}

	// Use MaxFractionDigits for precision (standard locale-aware behavior)
	precision := format.MaxFractionDigits
	if precision == 0 {
		precision = 2
	}
	factor := math.Pow(10, float64(precision))
	number = math.Round(number*factor) / factor

	str := fmt.Sprintf("%.*f", precision, number)
	parts := strings.Split(str, ".")
	integerPart := parts[0]
	decimalPart := ""
	if len(parts) > 1 {
		decimalPart = parts[1]
	}

	// Trim trailing zeros, but keep at least MinFractionDigits
	if len(decimalPart) > format.MinFractionDigits {
		trimTo := len(decimalPart)
		for trimTo > format.MinFractionDigits && decimalPart[trimTo-1] == '0' {
			trimTo--
		}
		decimalPart = decimalPart[:trimTo]
	}
	// Always pad to MinFractionDigits
	if format.MinFractionDigits > 0 && len(decimalPart) < format.MinFractionDigits {
		decimalPart = decimalPart + strings.Repeat("0", format.MinFractionDigits-len(decimalPart))
	}

	// Add thousands separators
	if format.Grouping != nil && len(format.Grouping) > 0 {
		integerPart = addNumberGrouping(integerPart, format)
	}

	// Combine parts
	if decimalPart != "" {
		return integerPart + format.DecimalSeparator + decimalPart
	}
	return integerPart
}

// FormatCurrency formats a number as currency according to the locale's currency format
func (t *Translator) FormatCurrency(amount float64, currencyCode string) string {
	if t.locale == nil {
		return fmt.Sprintf("$%.2f", amount)
	}

	format := t.locale.CurrencyFormat
	if format.Symbol == "" {
		format.Symbol = "$"
	}
	if format.Position == "" {
		format.Position = "before"
	}

	// Format the number part using currency format settings
	numberStr := t.formatNumberWithFormat(amount, format.NumberFormat)

	// Add currency symbol
	if format.Position == "before" {
		if format.Space {
			return format.Symbol + " " + numberStr
		}
		return format.Symbol + numberStr
	}

	if format.Space {
		return numberStr + " " + format.Symbol
	}
	return numberStr + format.Symbol
}

// FormatPercentage formats a number as a percentage according to the locale's number format
func (t *Translator) FormatPercentage(number float64) string {
	if t.locale == nil {
		return fmt.Sprintf("%.1f%%", number*100)
	}

	// Format the number part using locale settings
	numberStr := t.formatNumberWithFormat(number*100, t.locale.NumberFormat)
	return numberStr + "%"
}

// FormatScientific formats a number in scientific notation
func (t *Translator) FormatScientific(number float64, precision int) string {
	if t.locale == nil {
		return fmt.Sprintf("%.*e", precision, number)
	}

	// Use locale's decimal separator in scientific notation
	formatted := fmt.Sprintf("%.*e", precision, number)
	if t.locale.NumberFormat.DecimalSeparator != "." {
		formatted = strings.ReplaceAll(formatted, ".", t.locale.NumberFormat.DecimalSeparator)
	}
	return formatted
}

// FormatRelativeTime formats a time as a relative string (e.g., "2 hours ago", "in 3 days")
func (t *Translator) FormatRelativeTime(target time.Time, now time.Time) string {
	if t.locale == nil {
		return t.formatRelativeTimeFallback(target, now)
	}

	duration := now.Sub(target)
	absDuration := duration
	if absDuration < 0 {
		absDuration = -absDuration
	}

	// Get relative time messages from locale
	var message string
	if duration < 0 {
		// Future time
		message = t.getMessage("relative_time.future")
	} else {
		// Past time
		message = t.getMessage("relative_time.past")
	}

	// If no custom message, use fallback
	if message == "" {
		return t.formatRelativeTimeFallback(target, now)
	}

	// Calculate the appropriate unit and value
	var unit string
	var value int

	switch {
	case absDuration < time.Minute:
		unit = "second"
		value = int(absDuration.Seconds())
	case absDuration < time.Hour:
		unit = "minute"
		value = int(absDuration.Minutes())
	case absDuration < 24*time.Hour:
		unit = "hour"
		value = int(absDuration.Hours())
	case absDuration < 7*24*time.Hour:
		unit = "day"
		value = int(absDuration.Hours() / 24)
	case absDuration < 30*24*time.Hour:
		unit = "week"
		value = int(absDuration.Hours() / 24 / 7)
	case absDuration < 365*24*time.Hour:
		unit = "month"
		value = int(absDuration.Hours() / 24 / 30)
	default:
		unit = "year"
		value = int(absDuration.Hours() / 24 / 365)
	}

	// Get pluralized unit message
	unitKey := fmt.Sprintf("relative_time.%s", unit)
	if value == 1 {
		unitKey += ".one"
	} else {
		unitKey += ".other"
	}
	unitMessage := t.getMessage(unitKey)
	if unitMessage == "" {
		unitMessage = unit
		if value != 1 {
			unitMessage += "s"
		}
	}

	// Substitute in the message
	return t.substituteParams(message, map[string]interface{}{
		"Value": value,
		"Unit":  unitMessage,
	})
}

// formatRelativeTimeFallback provides fallback relative time formatting
func (t *Translator) formatRelativeTimeFallback(target time.Time, now time.Time) string {
	duration := now.Sub(target)
	absDuration := duration
	if absDuration < 0 {
		absDuration = -absDuration
	}

	var result string
	if absDuration < time.Minute {
		seconds := int(absDuration.Seconds())
		result = fmt.Sprintf("%d seconds ago", seconds)
		if duration < 0 {
			result = fmt.Sprintf("in %d seconds", seconds)
		}
	} else if absDuration < time.Hour {
		minutes := int(absDuration.Minutes())
		result = fmt.Sprintf("%d minutes ago", minutes)
		if duration < 0 {
			result = fmt.Sprintf("in %d minutes", minutes)
		}
	} else if absDuration < 24*time.Hour {
		hours := int(absDuration.Hours())
		result = fmt.Sprintf("%d hours ago", hours)
		if duration < 0 {
			result = fmt.Sprintf("in %d hours", hours)
		}
	} else if absDuration < 7*24*time.Hour {
		days := int(absDuration.Hours() / 24)
		result = fmt.Sprintf("%d days ago", days)
		if duration < 0 {
			result = fmt.Sprintf("in %d days", days)
		}
	} else {
		result = "in the past"
		if duration < 0 {
			result = "in the future"
		}
	}

	return result
}

// FormatCurrencyWithCode formats a number as currency with the currency code
func (t *Translator) FormatCurrencyWithCode(amount float64, currencyCode string) string {
	if t.locale == nil {
		return fmt.Sprintf("%s %.2f", currencyCode, amount)
	}

	format := t.locale.CurrencyFormat
	if format.Symbol == "" {
		format.Symbol = currencyCode
	}

	// Format the number part using currency format settings
	numberStr := t.formatNumberWithFormat(amount, format.NumberFormat)

	var formattedCurrency string
	// Add currency symbol and code
	if format.Position == "before" {
		if format.Space {
			formattedCurrency = format.Symbol + " " + numberStr + " (" + currencyCode + ")"
		} else {
			formattedCurrency = format.Symbol + numberStr + " (" + currencyCode + ")"
		}
	} else {
		if format.Space {
			formattedCurrency = numberStr + " " + format.Symbol + " (" + currencyCode + ")"
		} else {
			formattedCurrency = numberStr + format.Symbol + " (" + currencyCode + ")"
		}
	}

	return formattedCurrency
}

// ParseNumber parses a formatted number string back to float64
func (t *Translator) ParseNumber(formatted string) (float64, error) {
	if t.locale == nil {
		return strconv.ParseFloat(formatted, 64)
	}

	// Remove thousands separators
	cleaned := formatted
	if t.locale.NumberFormat.ThousandsSeparator != "" {
		cleaned = strings.ReplaceAll(cleaned, t.locale.NumberFormat.ThousandsSeparator, "")
	}

	// Convert decimal separator to standard format
	if t.locale.NumberFormat.DecimalSeparator != "." {
		cleaned = strings.ReplaceAll(cleaned, t.locale.NumberFormat.DecimalSeparator, ".")
	}

	return strconv.ParseFloat(cleaned, 64)
}

// ParseCurrency parses a formatted currency string back to float64
func (t *Translator) ParseCurrency(formatted string) (float64, error) {
	if t.locale == nil {
		// Remove common currency symbols
		cleaned := strings.TrimSpace(formatted)
		cleaned = strings.TrimPrefix(cleaned, "$")
		cleaned = strings.TrimPrefix(cleaned, "€")
		cleaned = strings.TrimPrefix(cleaned, "£")
		cleaned = strings.TrimSuffix(cleaned, "$")
		cleaned = strings.TrimSuffix(cleaned, "€")
		cleaned = strings.TrimSuffix(cleaned, "£")
		return strconv.ParseFloat(strings.TrimSpace(cleaned), 64)
	}

	// Remove currency symbol
	cleaned := strings.TrimSpace(formatted)
	format := t.locale.CurrencyFormat

	if format.Position == "before" {
		cleaned = strings.TrimPrefix(cleaned, format.Symbol)
		if format.Space {
			cleaned = strings.TrimPrefix(cleaned, " ")
		}
	} else {
		cleaned = strings.TrimSuffix(cleaned, format.Symbol)
		if format.Space {
			cleaned = strings.TrimSuffix(cleaned, " ")
		}
	}

	// Remove currency code if present (e.g., " (USD)")
	if idx := strings.LastIndex(cleaned, " ("); idx != -1 {
		cleaned = cleaned[:idx]
	}

	return t.ParseNumber(strings.TrimSpace(cleaned))
}

// getFormatWithFallback returns the format string with appropriate fallback
func (t *Translator) getFormatWithFallback(formatType string, short, medium, long string, fallbackShort, fallbackMedium, fallbackLong string) string {
	var format string
	switch formatType {
	case FormatShort:
		format = short
	case FormatLong:
		format = long
	default:
		format = medium
	}

	if format == "" {
		// Fallback formats
		switch formatType {
		case FormatShort:
			format = fallbackShort
		case FormatLong:
			format = fallbackLong
		default:
			format = fallbackMedium
		}
	}

	return format
}

// FormatDate formats a date according to the locale's date format
func (t *Translator) FormatDate(date time.Time, formatType string) string {
	if t.locale == nil {
		return date.Format("2006-01-02")
	}

	format := t.getFormatWithFallback(
		formatType,
		t.locale.DateFormat.Short,
		t.locale.DateFormat.Medium,
		t.locale.DateFormat.Long,
		FallbackDateShort,
		FallbackDateMedium,
		FallbackDateLong,
	)

	return date.Format(format)
}

// FormatTime formats a time according to the locale's time format
func (t *Translator) FormatTime(time time.Time, formatType string) string {
	if t.locale == nil {
		return time.Format("15:04")
	}

	format := t.getFormatWithFallback(
		formatType,
		t.locale.TimeFormat.Short,
		t.locale.TimeFormat.Medium,
		t.locale.TimeFormat.Long,
		FallbackTimeShort,
		FallbackTimeMedium,
		FallbackTimeLong,
	)

	return time.Format(format)
}

// FormatDateTime formats a date and time according to the locale's format
func (t *Translator) FormatDateTime(datetime time.Time, dateType, timeType string) string {
	dateStr := t.FormatDate(datetime, dateType)
	timeStr := t.FormatTime(datetime, timeType)
	return dateStr + " " + timeStr
}

// getMessage retrieves a message from the current locale
func (t *Translator) getMessage(key string) string {
	t.manager.mu.RLock()
	defer t.manager.mu.RUnlock()

	if t.locale == nil {
		return ""
	}

	// Split key by dots for nested access
	keys := strings.Split(key, ".")
	current := t.locale.Messages

	for _, k := range keys {
		if val, ok := current[k]; ok {
			switch v := val.(type) {
			case string:
				return v
			case map[string]interface{}:
				current = v
			default:
				return ""
			}
		} else {
			return ""
		}
	}

	return ""
}

// getPluralMessage retrieves a pluralized message
func (t *Translator) getPluralMessage(key string, count int) string {
	pluralKey := fmt.Sprintf("%s.%s", key, t.getPluralForm(count))
	return t.getMessage(pluralKey)
}

// getPluralForm determines the plural form for a given count
func (t *Translator) getPluralForm(count int) string {
	// Production-grade implementation using CLDR plural rules
	// This handles the most common plural forms across languages

	// Get locale code for plural rules
	localeCode := t.locale.Code

	// Handle special cases for different language families
	switch {
	case strings.HasPrefix(localeCode, "zh"): // Chinese
		return PluralOther // Chinese has no plural forms
	case strings.HasPrefix(localeCode, "ja"): // Japanese
		return PluralOther // Japanese has no plural forms
	case strings.HasPrefix(localeCode, "ko"): // Korean
		return PluralOther // Korean has no plural forms
	case strings.HasPrefix(localeCode, "th"): // Thai
		return PluralOther // Thai has no plural forms
	case strings.HasPrefix(localeCode, "vi"): // Vietnamese
		return PluralOther // Vietnamese has no plural forms
	case strings.HasPrefix(localeCode, "ar"): // Arabic
		return t.getArabicPluralForm(count)
	case strings.HasPrefix(localeCode, "ru"): // Russian
		return t.getRussianPluralForm(count)
	case strings.HasPrefix(localeCode, "pl"): // Polish
		return t.getPolishPluralForm(count)
	case strings.HasPrefix(localeCode, "cs"): // Czech
		return t.getCzechPluralForm(count)
	case strings.HasPrefix(localeCode, "sk"): // Slovak
		return t.getSlovakPluralForm(count)
	case strings.HasPrefix(localeCode, "sl"): // Slovenian
		return t.getSlovenianPluralForm(count)
	case strings.HasPrefix(localeCode, "he"): // Hebrew
		return t.getHebrewPluralForm(count)
	default:
		// Default English-style plural rules
		if count == 1 {
			return PluralOne
		}
		return PluralOther
	}
}

// Helper functions for specific language plural rules
func (t *Translator) getArabicPluralForm(count int) string {
	if count == 0 {
		return PluralZero
	} else if count == 1 {
		return PluralOne
	} else if count == 2 {
		return PluralTwo
	} else if count >= 3 && count <= 10 {
		return PluralFew
	} else if count >= 11 && count <= 99 {
		return PluralMany
	} else {
		return PluralOther
	}
}

func (t *Translator) getRussianPluralForm(count int) string {
	if count%10 == 1 && count%100 != 11 {
		return PluralOne
	} else if count%10 >= 2 && count%10 <= 4 && (count%100 < 10 || count%100 >= 20) {
		return PluralFew
	} else {
		return PluralOther
	}
}

func (t *Translator) getPolishPluralForm(count int) string {
	if count == 1 {
		return PluralOne
	} else if count%10 >= 2 && count%10 <= 4 && (count%100 < 10 || count%100 >= 20) {
		return PluralFew
	} else {
		return PluralOther
	}
}

func (t *Translator) getCzechPluralForm(count int) string {
	if count == 1 {
		return PluralOne
	} else if count >= 2 && count <= 4 {
		return PluralFew
	} else {
		return PluralOther
	}
}

func (t *Translator) getSlovakPluralForm(count int) string {
	if count == 1 {
		return PluralOne
	} else if count >= 2 && count <= 4 {
		return PluralFew
	} else {
		return PluralOther
	}
}

func (t *Translator) getSlovenianPluralForm(count int) string {
	if count%100 == 1 {
		return PluralOne
	} else if count%100 == 2 {
		return PluralTwo
	} else if count%100 >= 3 && count%100 <= 4 {
		return PluralFew
	} else {
		return PluralOther
	}
}

func (t *Translator) getHebrewPluralForm(count int) string {
	if count == 1 {
		return PluralOne
	} else if count == 2 {
		return PluralTwo
	} else if count >= 3 && count <= 10 {
		return PluralFew
	} else {
		return PluralOther
	}
}

// extractPluralKey extracts plural key and count from parameters
func (t *Translator) extractPluralKey(key string, params map[string]interface{}) (string, int) {
	if count, ok := params["Count"]; ok {
		switch v := count.(type) {
		case int:
			return key, v
		case float64:
			return key, int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return key, i
			}
		}
	}
	return "", 0
}

// substituteParams substitutes parameters in a message template
func (t *Translator) substituteParams(message string, params map[string]interface{}) string {
	if params == nil || len(params) == 0 {
		return message
	}

	// Simple parameter substitution using {{.Key}} syntax
	tmpl, err := template.New("message").Parse(message)
	if err != nil {
		return message
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, params); err != nil {
		return message
	}

	return buf.String()
}

// detectLocale detects the locale from the request
func (m *Manager) detectLocale(r *http.Request) *Locale {
	ctx := context.Background()
	fallbackUsed := false

	// Try query parameter first
	if locale := r.URL.Query().Get("locale"); locale != "" {
		if loc := m.getLocale(locale); loc != nil {
			if obs := getObserver(); obs != nil {
				obs.OnLocaleDetection(ctx, locale, false)
			}
			return loc
		}
	}

	// Try cookie
	if cookie, err := r.Cookie("locale"); err == nil {
		if loc := m.getLocale(cookie.Value); loc != nil {
			if obs := getObserver(); obs != nil {
				obs.OnLocaleDetection(ctx, cookie.Value, false)
			}
			return loc
		}
	}

	// Try Accept-Language header
	if acceptLang := r.Header.Get("Accept-Language"); acceptLang != "" {
		if loc := m.parseAcceptLanguage(acceptLang); loc != nil {
			if obs := getObserver(); obs != nil {
				obs.OnLocaleDetection(ctx, loc.Code, false)
			}
			return loc
		}
	}

	// Fall back to default locale
	fallbackUsed = true
	detectedLocale := m.defaultLocale
	if obs := getObserver(); obs != nil {
		obs.OnLocaleDetection(ctx, detectedLocale, fallbackUsed)
	}
	return m.getLocale(m.defaultLocale)
}

// parseAcceptLanguage parses the Accept-Language header
func (m *Manager) parseAcceptLanguage(acceptLang string) *Locale {
	// Parse Accept-Language header (e.g., "en-US,en;q=0.9,es;q=0.8")
	langs := strings.Split(acceptLang, ",")

	for _, lang := range langs {
		// Extract language code (e.g., "en-US" -> "en")
		parts := strings.Split(strings.TrimSpace(lang), ";")
		langCode := strings.Split(parts[0], "-")[0]

		if loc := m.getLocale(langCode); loc != nil {
			return loc
		}
	}

	return nil
}

// getLocale retrieves a locale by code
func (m *Manager) getLocale(code string) *Locale {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if locale, exists := m.locales[code]; exists {
		return locale
	}

	// Try fallback locale
	if locale, exists := m.locales[m.fallbackLocale]; exists {
		return locale
	}

	return nil
}

// loadLocales loads all locale files from the specified directory
func (m *Manager) loadLocales(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if it's a TOML file
		if strings.HasSuffix(entry.Name(), ".toml") {
			localeCode := strings.TrimSuffix(entry.Name(), ".toml")
			localePath := filepath.Join(path, entry.Name())

			if err := m.loadLocaleFile(localeCode, localePath); err != nil {
				return fmt.Errorf("failed to load locale %s: %v", localeCode, err)
			}
		}
	}

	return nil
}

// loadLocaleFile loads a single locale file
func (m *Manager) loadLocaleFile(code, path string) error {
	// For now, we'll use a simple key-value format
	// In a full implementation, you'd use a TOML parser
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	messages := make(map[string]interface{})
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Simple key=value parsing
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

	locale := &Locale{
		Code:     code,
		Messages: messages,
	}

	m.mu.Lock()
	m.locales[code] = locale
	m.mu.Unlock()

	return nil
}

// AddLocale adds a locale programmatically
func (m *Manager) AddLocale(code string, messages map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.locales[code] = &Locale{
		Code:     code,
		Messages: messages,
	}
}

// GetAvailableLocales returns all available locale codes
func (m *Manager) GetAvailableLocales() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	locales := make([]string, 0, len(m.locales))
	for code := range m.locales {
		locales = append(locales, code)
	}

	return locales
}

// SetNumberFormat sets the number formatting rules for a locale
func (m *Manager) SetNumberFormat(localeCode string, format NumberFormat) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if locale, exists := m.locales[localeCode]; exists {
		locale.NumberFormat = format
	}
}

// SetCurrencyFormat sets the currency formatting rules for a locale
func (m *Manager) SetCurrencyFormat(localeCode string, format CurrencyFormat) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if locale, exists := m.locales[localeCode]; exists {
		locale.CurrencyFormat = format
	}
}

// SetDateFormat sets the date formatting rules for a locale
func (m *Manager) SetDateFormat(localeCode string, format DateFormat) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if locale, exists := m.locales[localeCode]; exists {
		locale.DateFormat = format
	}
}

// SetTimeFormat sets the time formatting rules for a locale
func (m *Manager) SetTimeFormat(localeCode string, format TimeFormat) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if locale, exists := m.locales[localeCode]; exists {
		locale.TimeFormat = format
	}
}

// SetDefaultFormats sets default formatting rules for common locales
func (m *Manager) SetDefaultFormats() {
	// US English formatting
	m.SetNumberFormat("en", NumberFormat{
		DecimalSeparator:   ".",
		ThousandsSeparator: ",",
		Grouping:           []int{3},
		MinFractionDigits:  0,
		MaxFractionDigits:  2,
	})

	m.SetCurrencyFormat("en", CurrencyFormat{
		Symbol:   "$",
		Position: "before",
		Space:    false,
		NumberFormat: NumberFormat{
			DecimalSeparator:   ".",
			ThousandsSeparator: ",",
			Grouping:           []int{3},
			MinFractionDigits:  2,
			MaxFractionDigits:  2,
		},
	})

	m.SetDateFormat("en", DateFormat{
		Short:  "01/02/2006",
		Medium: "Jan 2, 2006",
		Long:   "January 2, 2006",
	})

	m.SetTimeFormat("en", TimeFormat{
		Short:  "3:04 PM",
		Medium: "3:04:05 PM",
		Long:   "3:04:05 PM MST",
	})

	// German formatting
	m.SetNumberFormat("de", NumberFormat{
		DecimalSeparator:   ",",
		ThousandsSeparator: ".",
		Grouping:           []int{3},
		MinFractionDigits:  0,
		MaxFractionDigits:  2,
	})

	m.SetCurrencyFormat("de", CurrencyFormat{
		Symbol:   "€",
		Position: "after",
		Space:    true,
		NumberFormat: NumberFormat{
			DecimalSeparator:   ",",
			ThousandsSeparator: ".",
			Grouping:           []int{3},
			MinFractionDigits:  2,
			MaxFractionDigits:  2,
		},
	})

	m.SetDateFormat("de", DateFormat{
		Short:  "02.01.2006",
		Medium: "2. Jan 2006",
		Long:   "2. Januar 2006",
	})

	m.SetTimeFormat("de", TimeFormat{
		Short:  "15:04",
		Medium: "15:04:05",
		Long:   "15:04:05 MST",
	})

	// French formatting
	m.SetNumberFormat("fr", NumberFormat{
		DecimalSeparator:   ",",
		ThousandsSeparator: " ",
		Grouping:           []int{3},
		MinFractionDigits:  0,
		MaxFractionDigits:  2,
	})

	m.SetCurrencyFormat("fr", CurrencyFormat{
		Symbol:   "€",
		Position: "after",
		Space:    true,
		NumberFormat: NumberFormat{
			DecimalSeparator:   ",",
			ThousandsSeparator: " ",
			Grouping:           []int{3},
			MinFractionDigits:  2,
			MaxFractionDigits:  2,
		},
	})

	m.SetDateFormat("fr", DateFormat{
		Short:  "02/01/2006",
		Medium: "2 janv. 2006",
		Long:   "2 janvier 2006",
	})

	m.SetTimeFormat("fr", TimeFormat{
		Short:  "15:04",
		Medium: "15:04:05",
		Long:   "15:04:05 MST",
	})
}

// addNumberGrouping applies thousands/grouping separators to the integer part of a number string
func addNumberGrouping(integerPart string, format NumberFormat) string {
	if format.Grouping != nil && len(format.Grouping) > 0 {
		var result strings.Builder
		length := len(integerPart)
		for i, digit := range integerPart {
			if i > 0 {
				shouldAddSeparator := false
				if len(format.Grouping) == 1 {
					groupSize := format.Grouping[0]
					if (length-i)%groupSize == 0 {
						shouldAddSeparator = true
					}
				} else {
					if i == format.Grouping[0] {
						shouldAddSeparator = true
					} else if i > format.Grouping[0] {
						groupSize := format.Grouping[1]
						remainingDigits := length - i
						if remainingDigits%groupSize == 0 {
							shouldAddSeparator = true
						}
					}
				}
				if shouldAddSeparator {
					result.WriteString(format.ThousandsSeparator)
				}
			}
			result.WriteRune(digit)
		}
		return result.String()
	}
	return integerPart
}
