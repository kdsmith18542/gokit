// Package form provides a comprehensive form validation and sanitization library for Go web applications.
//
// Features:
//   - Declarative validation using struct tags
//   - Built-in validators (required, email, min, max, url, etc.)
//   - Advanced conditional validation (required_if, eqfield, gtfield, ltfield)
//   - Custom validators with context support
//   - Input sanitization (trim, escape_html, to_lower, etc.)
//   - Observability hooks for tracing and metrics
//   - Support for both regular forms and multipart file uploads
//
// Example:
//
//	type SignUpForm struct {
//	    Email           string `form:"email" validate:"required,email"`
//	    Password        string `form:"password" validate:"required,min=8"`
//	    ConfirmPassword string `form:"confirm_password" validate:"required,eqfield=Password"`
//	    Bio             string `form:"bio" sanitize:"trim,escape_html"`
//	}
//
//	func SignUpHandler(w http.ResponseWriter, r *http.Request) {
//	    var s SignUpForm
//	    errs := form.DecodeAndValidate(r, &s)
//	    if errs != nil {
//	        // Handle validation errors
//	        return
//	    }
//	    // Process valid form data
//	}
package form

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/kdsmith18542/gokit/observability"
)

// Pre-compiled regular expressions for validators and sanitizers
// These are compiled once at package load time for better performance
var (
	emailRegex      = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	urlRegex        = regexp.MustCompile(`^https?://[^\s/$.?#].\S*$`)
	dateRegex       = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	whitespaceRegex = regexp.MustCompile(`\s+`)
	snakeKebabRegex = regexp.MustCompile(`[^a-z0-9]+`)
	htmlTagRegex    = regexp.MustCompile(`<[^>]*>`)
)

// ValidationErrors represents a map of field names to their validation error messages.
// Each field can have multiple validation errors.
type ValidationErrors map[string][]string

// Validator is a function that validates a value and returns an error message if invalid.
// If the value is valid, return an empty string.
//
// Example:
//
//	form.RegisterValidator("not_foo", func(value string) string {
//	    if value == "foo" {
//	        return "Value cannot be 'foo'"
//	    }
//	    return ""
//	})
type Validator func(value string) string

// ContextValidator is a function that validates a value with access to the full form context.
// This allows for cross-field validation and complex validation logic.
// If the value is valid, return an empty string.
//
// Example (DB uniqueness check):
//
//	form.RegisterContextValidator("unique_email", func(value, param string, ctx form.ValidationContext) string {
//	    // Simulate DB check (replace with real DB call)
//	    if value == "taken@example.com" {
//	        return "Email is already registered"
//	    }
//	    return ""
//	})
//
// Usage:
//
//	type SignUpForm struct {
//	    Email string `form:"email" validate:"required,email,unique_email"`
//	}
type ContextValidator func(value, param string, context ValidationContext) string

// Sanitizer is a function that sanitizes a value and returns the sanitized version.
// Sanitizers are applied before validation and can transform the input.
type Sanitizer func(value string) string

// ValidationContext provides access to all form field values for cross-field validation.
// Use this in custom validators that need to compare or reference other fields.
type ValidationContext struct {
	values map[string]string
}

// Get returns the value of a field by name.
// Returns an empty string if the field is not found.
// This method handles both form tags and field names by trying multiple variations.
func (c ValidationContext) Get(fieldName string) string {
	// Try exact match first
	if value, exists := c.values[fieldName]; exists {
		return value
	}

	// Try lowercase version
	if value, exists := c.values[strings.ToLower(fieldName)]; exists {
		return value
	}

	// Try common variations for cross-field validation
	variations := []string{
		fieldName,
		strings.ToLower(fieldName),
		strings.ReplaceAll(fieldName, "_", ""),
		strings.ReplaceAll(strings.ToLower(fieldName), "_", ""),
	}

	for _, variation := range variations {
		if value, exists := c.values[variation]; exists {
			return value
		}
	}

	return ""
}

// Registry holds all registered validators and sanitizers.
// The package maintains a global registry instance that can be extended with custom validators and sanitizers.
type Registry struct {
	validators        map[string]Validator
	contextValidators map[string]ContextValidator
	sanitizers        map[string]Sanitizer
}

// Global registry instance
var registry = &Registry{
	validators:        make(map[string]Validator),
	contextValidators: make(map[string]ContextValidator),
	sanitizers:        make(map[string]Sanitizer),
}

// RegisterValidator registers a custom validator function.
// The validator will be available for use in struct tags.
//
// Example:
//
//	form.RegisterValidator("custom_rule", func(value string) string {
//	    if value == "invalid" {
//	        return "Value cannot be 'invalid'"
//	    }
//	    return ""
//	})
func RegisterValidator(name string, validator Validator) {
	registry.validators[name] = validator
}

// RegisterContextValidator registers a custom context-aware validator function.
// This validator has access to all form field values for complex validation logic.
//
// Example (cross-field and DB check):
//
//	form.RegisterContextValidator("unique_username", func(value, param string, ctx form.ValidationContext) string {
//	    // Simulate DB uniqueness check
//	    if value == "admin" {
//	        return "Username is reserved"
//	    }
//	    return ""
//	})
//
//	form.RegisterContextValidator("not_equal", func(value, param string, ctx form.ValidationContext) string {
//	    if value == ctx.Get(param) {
//	        return "Fields must not match"
//	    }
//	    return ""
//	})
func RegisterContextValidator(name string, validator ContextValidator) {
	registry.contextValidators[name] = validator
}

// RegisterSanitizer registers a custom sanitizer function.
// Sanitizers are applied before validation and can transform input values.
//
// Example:
//
//	form.RegisterSanitizer("remove_spaces", func(value string) string {
//	    return strings.ReplaceAll(value, " ", "")
//	})
func RegisterSanitizer(name string, sanitizer Sanitizer) {
	registry.sanitizers[name] = sanitizer
}

// DecodeAndValidate decodes form data from an HTTP request and validates it against a struct.
// This is the main function for form processing. It handles both regular forms and multipart file uploads.
//
// The target struct should have form tags to specify field names and validation rules:
//   - `form:"field_name"` - specifies the form field name
//   - `validate:"rule1,rule2"` - specifies validation rules
//   - `sanitize:"sanitizer1,sanitizer2"` - specifies sanitization rules
//
// Returns a ValidationErrors map. If the map is empty, validation passed.
func DecodeAndValidate(r *http.Request, v interface{}) ValidationErrors {
	return DecodeAndValidateWithContext(context.Background(), r, v)
}

// DecodeAndValidateWithContext decodes form data from an HTTP request and validates it against a struct with context.
// This version accepts a context.Context for observability and cancellation support.
//
// The context is passed to any registered observers for tracing and metrics, and is available to context-aware validators.
//
// Example (with context-aware validator):
//
//	form.RegisterContextValidator("unique_email", func(value, param string, ctx form.ValidationContext) string {
//	    // Use ctx.Context for DB/API calls if needed
//	    // ...
//	    return ""
//	})
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    var f SignUpForm
//	    errs := form.DecodeAndValidateWithContext(r.Context(), r, &f)
//	    if len(errs) > 0 {
//	        // Handle errors
//	    }
//	    // Use validated form
//	}
func DecodeAndValidateWithContext(ctx context.Context, r *http.Request, v interface{}) ValidationErrors {
	start := time.Now()
	formName := ""
	if v != nil {
		formName = reflect.TypeOf(v).Elem().Name()
	}
	if obs := getObserver(); obs != nil {
		obs.OnDecodeStart(ctx, formName)
	}
	errors := make(ValidationErrors)

	// Parse form data
	if err := r.ParseForm(); err != nil {
		errors["_form"] = []string{"Failed to parse form data"}
		if obs := getObserver(); obs != nil {
			obs.OnDecodeEnd(ctx, formName, err)
		}
		return errors
	}

	// Parse multipart form if needed
	contentType := r.Header.Get("Content-Type")
	if r.MultipartForm == nil && strings.Contains(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			errors["_form"] = []string{"Failed to parse multipart form data"}
			if obs := getObserver(); obs != nil {
				obs.OnDecodeEnd(ctx, formName, err)
			}
			return errors
		}
	}

	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		errors["_struct"] = []string{"Target must be a non-nil pointer to struct"}
		if obs := getObserver(); obs != nil {
			obs.OnDecodeEnd(ctx, formName, nil)
		}
		return errors
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		errors["_struct"] = []string{"Target must be a pointer to struct"}
		if obs := getObserver(); obs != nil {
			obs.OnDecodeEnd(ctx, formName, nil)
		}
		return errors
	}

	// Convert request data to form-like structure
	formData := make(map[string][]string)
	if r.MultipartForm != nil {
		for key, values := range r.MultipartForm.Value {
			formData[key] = values
		}
	} else {
		for key, values := range r.Form {
			formData[key] = values
		}
	}

	// First pass: collect all field values and apply sanitizers
	fieldValues := processFormFields(val, formData)

	if obs := getObserver(); obs != nil {
		obs.OnDecodeEnd(ctx, formName, nil)
		obs.OnValidationStart(ctx, formName)
	}

	// Second pass: validate fields
	validationErrors := validateFormFields(val, fieldValues)

	handleFormObservability(ctx, formName, validationErrors, start)

	return validationErrors
}

// applySanitizers applies a chain of sanitizers to a value
func applySanitizers(value, sanitizeTag string) string {
	sanitizers := strings.Split(sanitizeTag, ",")
	for _, sanitizerName := range sanitizers {
		sanitizerName = strings.TrimSpace(sanitizerName)
		if sanitizer, exists := registry.sanitizers[sanitizerName]; exists {
			value = sanitizer(value)
		}
	}
	return value
}

// validateFieldWithContext validates a field value against validation rules with context
func validateFieldWithContext(value, validateTag string, context ValidationContext, kind ...reflect.Kind) []string {
	var errors []string
	validators := strings.Split(validateTag, ",")
	var fieldKind reflect.Kind
	if len(kind) > 0 {
		fieldKind = kind[0]
	}

	for _, validatorRule := range validators {
		validatorRule = strings.TrimSpace(validatorRule)
		parts := strings.SplitN(validatorRule, "=", 2)
		validatorName := parts[0]
		var param string
		if len(parts) > 1 {
			param = parts[1]
		}

		// Check context validators first (for cross-field validation)
		if contextValidator, exists := registry.contextValidators[validatorName]; exists {
			if errorMsg := contextValidator(value, param, context); errorMsg != "" {
				errors = append(errors, errorMsg)
			}
		} else if validator, exists := registry.validators[validatorName]; exists {
			if errorMsg := validator(value); errorMsg != "" {
				errors = append(errors, errorMsg)
			}
		} else if builtinValidator, exists := builtinValidators[validatorName]; exists {
			if validatorName == "min" || validatorName == "max" {
				if errorMsg := builtinValidatorWithKind(value, param, fieldKind, validatorName); errorMsg != "" {
					errors = append(errors, errorMsg)
				}
			} else {
				if errorMsg := builtinValidator(value, param); errorMsg != "" {
					errors = append(errors, errorMsg)
				}
			}
		} else if builtinContextValidator, exists := builtinContextValidators[validatorName]; exists {
			if errorMsg := builtinContextValidator(value, param, context); errorMsg != "" {
				errors = append(errors, errorMsg)
			}
		}
	}

	return errors
}

// isNumericType checks if a reflect.Kind represents a numeric type
func isNumericType(kind reflect.Kind) bool {
	return kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 ||
		kind == reflect.Int32 || kind == reflect.Int64 || kind == reflect.Uint ||
		kind == reflect.Uint8 || kind == reflect.Uint16 || kind == reflect.Uint32 ||
		kind == reflect.Uint64 || kind == reflect.Float32 || kind == reflect.Float64
}

// builtinValidatorWithKind handles min/max with type awareness
func builtinValidatorWithKind(value, param string, kind reflect.Kind, validatorName string) string {
	if value == "" {
		return ""
	}

	if validatorName == "min" {
		return validateMin(value, param, kind)
	} else if validatorName == "max" {
		return validateMax(value, param, kind)
	}
	return ""
}

// validateMin validates minimum value constraints
func validateMin(value, param string, kind reflect.Kind) string {
	minVal, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return ""
	}

	if isNumericType(kind) {
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return ErrMustBeNumber
		}
		if num < minVal {
			return fmt.Sprintf("Must be at least %v", param)
		}
		return ""
	}

	// fallback to string length
	if len(value) < int(minVal) {
		return fmt.Sprintf("Must be at least %d characters long", int(minVal))
	}
	return ""
}

// validateMax validates maximum value constraints
func validateMax(value, param string, kind reflect.Kind) string {
	maxVal, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return ""
	}

	if isNumericType(kind) {
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return ErrMustBeNumber
		}
		if num > maxVal {
			return fmt.Sprintf("Must be no more than %v", param)
		}
		return ""
	}

	// fallback to string length
	if len(value) > int(maxVal) {
		return fmt.Sprintf("Must be no more than %d characters long", int(maxVal))
	}
	return ""
}

// setFieldValue sets a field value based on its type
func setFieldValue(field reflect.Value, value string) {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value != "" {
			if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
				field.SetInt(intVal)
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value != "" {
			if uintVal, err := strconv.ParseUint(value, 10, 64); err == nil {
				field.SetUint(uintVal)
			}
		}
	case reflect.Float32, reflect.Float64:
		if value != "" {
			if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
				field.SetFloat(floatVal)
			}
		}
	case reflect.Bool:
		if value != "" {
			if boolVal, err := strconv.ParseBool(value); err == nil {
				field.SetBool(boolVal)
			}
		}
	}
}

// builtinValidators contains all built-in validation functions
var builtinValidators = map[string]func(value, param string) string{
	"required": func(value, param string) string {
		if strings.TrimSpace(value) == "" {
			return ErrFieldRequired
		}
		return ""
	},
	"email": func(value, param string) string {
		if value == "" {
			return ""
		}
		if !emailRegex.MatchString(value) {
			return ErrInvalidEmail
		}
		return ""
	},
	"min": func(value, param string) string {
		if value == "" {
			return ""
		}
		minVal, err := strconv.Atoi(param)
		if err != nil {
			return ""
		}
		if len(value) < minVal {
			return fmt.Sprintf("Must be at least %d characters long", minVal)
		}
		return ""
	},
	"max": func(value, param string) string {
		if value == "" {
			return ""
		}
		maxVal, err := strconv.Atoi(param)
		if err != nil {
			return ""
		}
		if len(value) > maxVal {
			return fmt.Sprintf("Must be no more than %d characters long", maxVal)
		}
		return ""
	},
	"url": func(value, param string) string {
		if value == "" {
			return ""
		}
		if !urlRegex.MatchString(value) {
			return ErrInvalidURL
		}
		return ""
	},
	"numeric": func(value, param string) string {
		if value == "" {
			return ""
		}
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			return ""
		}
		return ErrMustBeNumber
	},
	"alpha": func(value, param string) string {
		if value == "" {
			return ""
		}
		for _, char := range value {
			if !unicode.IsLetter(char) {
				return ErrMustBeAlpha
			}
		}
		return ""
	},
	"alphanumeric": func(value, param string) string {
		if value == "" {
			return ""
		}
		for _, char := range value {
			if !unicode.IsLetter(char) && !unicode.IsNumber(char) {
				return ErrMustBeAlphanumeric
			}
		}
		return ""
	},
}

// builtinContextValidators contains all built-in context-aware validation functions
var builtinContextValidators = map[string]ContextValidator{
	"required_if": func(value, param string, context ValidationContext) string {
		// required_if=field:value means this field is required if the specified field equals the specified value
		// required_if=field means this field is required if the specified field is not empty
		if strings.TrimSpace(value) == "" {
			if strings.Contains(param, ":") {
				// Parse field:value format
				parts := strings.SplitN(param, ":", 2)
				if len(parts) == 2 {
					fieldName := parts[0]
					expectedValue := parts[1]
					actualValue := context.Get(fieldName)
					if actualValue == expectedValue {
						return ErrFieldRequired
					}
				}
			} else {
				// Check if the specified field is not empty
				otherValue := context.Get(param)
				if strings.TrimSpace(otherValue) != "" {
					return ErrFieldRequired
				}
			}
		}
		return ""
	},
	"required_unless": func(value, param string, context ValidationContext) string {
		// required_unless=field:value means this field is required unless the specified field equals the specified value
		if strings.TrimSpace(value) == "" {
			if strings.Contains(param, ":") {
				parts := strings.SplitN(param, ":", 2)
				if len(parts) == 2 {
					fieldName := parts[0]
					exemptValue := parts[1]
					actualValue := context.Get(fieldName)
					if actualValue != exemptValue {
						return ErrFieldRequired
					}
				}
			} else {
				// If no value specified, check if the field is empty
				otherValue := context.Get(param)
				if strings.TrimSpace(otherValue) == "" {
					return ErrFieldRequired
				}
			}
		}
		return ""
	},
	"eqfield": func(value, param string, context ValidationContext) string {
		// eqfield=fieldname means this field must equal the specified field
		if param == "" {
			return ""
		}
		otherValue := context.Get(param)
		if value != otherValue {
			return fmt.Sprintf("Must match the value of %q", param)
		}
		return ""
	},
	"nefield": func(value, param string, context ValidationContext) string {
		// nefield=fieldname means this field must not equal the specified field
		if param == "" {
			return ""
		}
		otherValue := context.Get(param)
		if value == otherValue {
			return fmt.Sprintf("Must not match the value of %q", param)
		}
		return ""
	},
	"gtfield": func(value, param string, context ValidationContext) string {
		// gtfield=fieldname means this field must be greater than the specified field
		if value == "" || param == "" {
			return ""
		}
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return ErrMustBeNumber
		}
		otherValue := context.Get(param)
		if otherValue != "" {
			otherVal, err := strconv.ParseFloat(otherValue, 64)
			if err == nil && val <= otherVal {
				return fmt.Sprintf("Must be greater than %q", param)
			}
		}
		return ""
	},
	"gtefield": func(value, param string, context ValidationContext) string {
		// gtefield=fieldname means this field must be greater than or equal to the specified field
		if value == "" || param == "" {
			return ""
		}
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return ErrMustBeNumber
		}
		otherValue := context.Get(param)
		if otherValue != "" {
			otherVal, err := strconv.ParseFloat(otherValue, 64)
			if err == nil && val < otherVal {
				return fmt.Sprintf("Must be greater than or equal to %q", param)
			}
		}
		return ""
	},
	"ltfield": func(value, param string, context ValidationContext) string {
		// ltfield=fieldname means this field must be less than the specified field
		if value == "" || param == "" {
			return ""
		}
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return ErrMustBeNumber
		}
		otherValue := context.Get(param)
		if otherValue != "" {
			otherVal, err := strconv.ParseFloat(otherValue, 64)
			if err == nil && val >= otherVal {
				return fmt.Sprintf("Must be less than %q", param)
			}
		}
		return ""
	},
	"ltefield": func(value, param string, context ValidationContext) string {
		// ltefield=fieldname means this field must be less than or equal to the specified field
		if value == "" || param == "" {
			return ""
		}
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return ErrMustBeNumber
		}
		otherValue := context.Get(param)
		if otherValue != "" {
			otherVal, err := strconv.ParseFloat(otherValue, 64)
			if err == nil && val > otherVal {
				return fmt.Sprintf("Must be less than or equal to %q", param)
			}
		}
		return ""
	},
	"date_after": func(value, param string, context ValidationContext) string {
		// date_after=fieldname means this date must be after the specified field's date
		if value == "" || param == "" {
			return ""
		}
		// Parse dates (assuming YYYY-MM-DD format for simplicity)
		if !dateRegex.MatchString(value) {
			return "Must be a valid date (YYYY-MM-DD)"
		}
		otherValue := context.Get(param)
		if otherValue != "" && dateRegex.MatchString(otherValue) {
			if value <= otherValue {
				return fmt.Sprintf("Must be after %q", param)
			}
		}
		return ""
	},
	"date_before": func(value, param string, context ValidationContext) string {
		// date_before=fieldname means this date must be before the specified field's date
		if value == "" || param == "" {
			return ""
		}
		// Parse dates (assuming YYYY-MM-DD format for simplicity)
		if !dateRegex.MatchString(value) {
			return "Must be a valid date (YYYY-MM-DD)"
		}
		otherValue := context.Get(param)
		if otherValue != "" && dateRegex.MatchString(otherValue) {
			if value >= otherValue {
				return fmt.Sprintf("Must be before %q", param)
			}
		}
		return ""
	},
}

// builtinSanitizers contains all built-in sanitization functions
var builtinSanitizers = map[string]Sanitizer{
	"trim":        strings.TrimSpace,
	"to_lower":    strings.ToLower,
	"to_upper":    strings.ToUpper,
	"escape_html": html.EscapeString,
	"strip_numeric": func(value string) string {
		var result strings.Builder
		for _, char := range value {
			if !unicode.IsDigit(char) {
				result.WriteRune(char)
			}
		}
		return result.String()
	},
	"strip_alpha": func(value string) string {
		var result strings.Builder
		for _, char := range value {
			if !unicode.IsLetter(char) {
				result.WriteRune(char)
			}
		}
		return result.String()
	},
	"normalize_whitespace": func(value string) string {
		// Replace multiple whitespace characters with a single space
		return whitespaceRegex.ReplaceAllString(value, " ")
	},
	"remove_special_chars": func(value string) string {
		var result strings.Builder
		for _, char := range value {
			if unicode.IsLetter(char) || unicode.IsDigit(char) {
				result.WriteRune(char)
			} else if unicode.IsSpace(char) {
				result.WriteRune(char)
			} else {
				// Replace special characters with space
				result.WriteRune(' ')
			}
		}
		return result.String()
	},
	"title_case": func(value string) string {
		if value == "" {
			return value
		}
		words := strings.Fields(strings.ToLower(value))
		if len(words) == 0 {
			return value
		}
		// Capitalize first letter of each word
		for i := range words {
			if words[i] != "" {
				words[i] = strings.ToUpper(words[i][:1]) + words[i][1:]
			}
		}
		return strings.Join(words, " ")
	},
	"camel_case": func(value string) string {
		words := strings.Fields(strings.ToLower(value))
		if len(words) == 0 {
			return ""
		}

		var result strings.Builder
		result.WriteString(words[0]) // First word lowercase

		for i := 1; i < len(words); i++ {
			if words[i] != "" {
				result.WriteString(strings.ToUpper(words[i][:1]) + words[i][1:])
			}
		}
		return result.String()
	},
	"snake_case": func(value string) string {
		// Convert to lowercase and replace spaces/special chars with underscores
		value = strings.ToLower(value)
		value = snakeKebabRegex.ReplaceAllString(value, "_")
		return strings.Trim(value, "_")
	},
	"kebab_case": func(value string) string {
		// Convert to lowercase and replace spaces/special chars with hyphens
		value = strings.ToLower(value)
		value = snakeKebabRegex.ReplaceAllString(value, "-")
		return strings.Trim(value, "-")
	},
	"remove_html_tags": func(value string) string {
		// Simple HTML tag removal (for more complex cases, consider using a proper HTML parser)
		return htmlTagRegex.ReplaceAllString(value, "")
	},
	"normalize_unicode": func(value string) string {
		// Normalize unicode characters (NFD form)
		return strings.ToValidUTF8(value, "")
	},
}

// Initialize built-in sanitizers and context validators
func init() {
	// Register default built-in validators using pre-compiled regex
	RegisterValidator("required", func(value string) string {
		if strings.TrimSpace(value) == "" {
			return ErrFieldRequired
		}
		return ""
	})

	RegisterValidator("email", func(value string) string {
		if value == "" {
			return ""
		}
		if !emailRegex.MatchString(value) {
			return ErrInvalidEmail
		}
		return ""
	})

	RegisterValidator("url", func(value string) string {
		if value == "" {
			return ""
		}
		if !urlRegex.MatchString(value) {
			return ErrInvalidURL
		}
		return ""
	})

	RegisterContextValidator("eqfield", func(value, param string, ctx ValidationContext) string {
		if ctx.Get(param) != value {
			return fmt.Sprintf("Must match the %q field", param)
		}
		return ""
	})

	RegisterContextValidator("required_if", func(value, param string, ctx ValidationContext) string {
		parts := strings.Split(param, ":")
		if len(parts) != 2 {
			return "Invalid parameter for required_if"
		}
		fieldName := parts[0]
		expectedValue := parts[1]
		if ctx.Get(fieldName) == expectedValue && strings.TrimSpace(value) == "" {
			return fmt.Sprintf("This field is required when %s is %s", fieldName, expectedValue)
		}
		return ""
	})

	RegisterContextValidator("gtfield", func(value, param string, ctx ValidationContext) string {
		otherValue := ctx.Get(param)
		if value == "" || otherValue == "" {
			return ""
		}
		val1, err1 := strconv.ParseFloat(value, 64)
		val2, err2 := strconv.ParseFloat(otherValue, 64)
		if err1 != nil || err2 != nil {
			return ErrMustBeNumber
		}
		if val1 <= val2 {
			return fmt.Sprintf("Must be greater than %q", param)
		}
		return ""
	})

	RegisterContextValidator("ltfield", func(value, param string, ctx ValidationContext) string {
		otherValue := ctx.Get(param)
		if value == "" || otherValue == "" {
			return ""
		}
		val1, err1 := strconv.ParseFloat(value, 64)
		val2, err2 := strconv.ParseFloat(otherValue, 64)
		if err1 != nil || err2 != nil {
			return ErrMustBeNumber
		}
		if val1 >= val2 {
			return fmt.Sprintf("Must be less than %q", param)
		}
		return ""
	})

	// Register all builtin sanitizers
	for name, sanitizer := range builtinSanitizers {
		RegisterSanitizer(name, sanitizer)
	}

	RegisterValidator("is_uppercase", func(value string) string {
		if value == "" {
			return ""
		}
		for _, r := range value {
			if !unicode.IsUpper(r) && unicode.IsLetter(r) {
				return "Must be all uppercase"
			}
		}
		return ""
	})

	RegisterContextValidator("unique_username", func(value, param string, ctx ValidationContext) string {
		// Simulate an asynchronous database check
		// In a real application, this would involve a database query
		if value == "admin" || value == "testuser" {
			return "Username already taken"
		}
		return ""
	})

	// Initialize the default observer to nil, users can set their own
	// through observability.SetObserver. This avoids a global default
	// that might not be desired.
	observability.SetObserver(nil)
}
