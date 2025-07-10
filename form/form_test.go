package form

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// createRequest creates a test HTTP request with form data
func createRequest(data map[string]string) *http.Request {
	values := url.Values{}
	for key, value := range data {
		values.Set(key, value)
	}

	req, _ := http.NewRequest("POST", "/test", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

// TestForm represents a test form structure
type TestForm struct {
	Email    string `form:"email" validate:"required,email"`
	Password string `form:"password" validate:"required,min=8"`
	Age      int    `form:"age" validate:"numeric"`
	Name     string `form:"name" sanitize:"trim,to_lower"`
	Bio      string `form:"bio" sanitize:"trim,escape_html"`
	Website  string `form:"website" validate:"url"`
	Username string `form:"username" validate:"alphanumeric"`
}

// TestOptionalForm represents a form with optional fields
type TestOptionalForm struct {
	Email    string `form:"email" validate:"email"`
	Password string `form:"password" validate:"min=8"`
	Age      int    `form:"age"`
}

func TestDecodeAndValidate_ValidForm(t *testing.T) {
	form := url.Values{}
	form.Set("email", TestEmail)
	form.Set("password", "password123")
	form.Set("age", "25")
	form.Set("name", "  JOHN DOE  ")
	form.Set("bio", "<script>alert('xss')</script>")
	form.Set("website", "https://example.com")
	form.Set("username", "john123")

	req, _ := http.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var testForm TestForm
	errors := DecodeAndValidate(req, &testForm)

	if len(errors) > 0 {
		t.Errorf("Expected no validation errors, got: %v", errors)
	}

	if testForm.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", testForm.Email)
	}

	if testForm.Password != "password123" {
		t.Errorf("Expected password 'password123', got '%s'", testForm.Password)
	}

	if testForm.Age != 25 {
		t.Errorf("Expected age 25, got %d", testForm.Age)
	}

	// Check sanitization
	if testForm.Name != "john doe" {
		t.Errorf("Expected name 'john doe', got '%s'", testForm.Name)
	}

	if testForm.Bio != "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;" {
		t.Errorf("Expected bio to be HTML escaped, got '%s'", testForm.Bio)
	}
}

func TestDecodeAndValidate_InvalidEmail(t *testing.T) {
	form := url.Values{}
	form.Set("email", "invalid-email")
	form.Set("password", "password123")
	form.Set("age", "25")

	req, _ := http.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var testForm TestForm
	errors := DecodeAndValidate(req, &testForm)

	if len(errors) == 0 {
		t.Error("Expected validation errors for invalid email")
	}

	if errors["email"] == nil {
		t.Error("Expected email validation errors")
	}
}

func TestDecodeAndValidate_ShortPassword(t *testing.T) {
	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("password", "123")
	form.Set("age", "25")

	req, _ := http.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var testForm TestForm
	errors := DecodeAndValidate(req, &testForm)

	if len(errors) == 0 {
		t.Error("Expected validation errors for short password")
	}

	if errors["password"] == nil {
		t.Error("Expected password validation errors")
	}
}

func TestDecodeAndValidate_RequiredFields(t *testing.T) {
	form := url.Values{}
	form.Set("age", "25")

	req, _ := http.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var testForm TestForm
	errors := DecodeAndValidate(req, &testForm)

	if len(errors) == 0 {
		t.Error("Expected validation errors for missing required fields")
	}

	if errors["email"] == nil {
		t.Error("Expected email required validation error")
	}

	if errors["password"] == nil {
		t.Error("Expected password required validation error")
	}
}

func TestDecodeAndValidate_OptionalFields(t *testing.T) {
	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("password", "password123")

	req, _ := http.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var testForm TestOptionalForm
	errors := DecodeAndValidate(req, &testForm)

	if len(errors) > 0 {
		t.Errorf("Expected no validation errors for optional fields, got: %v", errors)
	}
}

func TestDecodeAndValidate_InvalidURL(t *testing.T) {
	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("password", "password123")
	form.Set("age", "25")
	form.Set("website", "not-a-url")

	req, _ := http.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var testForm TestForm
	errors := DecodeAndValidate(req, &testForm)

	if len(errors) == 0 {
		t.Error("Expected validation errors for invalid URL")
	}

	if errors["website"] == nil {
		t.Error("Expected website validation errors")
	}
}

func TestDecodeAndValidate_InvalidUsername(t *testing.T) {
	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("password", "password123")
	form.Set("age", "25")
	form.Set("username", "john@123")

	req, _ := http.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var testForm TestForm
	errors := DecodeAndValidate(req, &testForm)

	if len(errors) == 0 {
		t.Error("Expected validation errors for invalid username")
	}

	if errors["username"] == nil {
		t.Error("Expected username validation errors")
	}
}

func TestDecodeAndValidate_MultipartForm(t *testing.T) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writer.WriteField("email", "test@example.com"); err != nil {
		t.Fatalf("Failed to write email field: %v", err)
	}
	if err := writer.WriteField("password", "password123"); err != nil {
		t.Fatalf("Failed to write password field: %v", err)
	}
	if err := writer.WriteField("age", "25"); err != nil {
		t.Fatalf("Failed to write age field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	req, _ := http.NewRequest("POST", "/test", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	var testForm TestForm
	errors := DecodeAndValidate(req, &testForm)

	if len(errors) > 0 {
		t.Errorf("Expected no validation errors, got: %v", errors)
	}

	if testForm.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", testForm.Email)
	}
}

func TestDecodeAndValidate_InvalidStruct(t *testing.T) {
	req, _ := http.NewRequest("POST", "/test", strings.NewReader(""))

	// Test with nil pointer
	var testForm *TestForm
	errors := DecodeAndValidate(req, testForm)

	if len(errors) == 0 {
		t.Error("Expected validation errors for nil pointer")
	}

	if errors["_struct"] == nil {
		t.Error("Expected struct validation errors")
	}
}

func TestDecodeAndValidate_NonStruct(t *testing.T) {
	req, _ := http.NewRequest("POST", "/test", strings.NewReader(""))

	// Test with non-struct
	var str string
	errors := DecodeAndValidate(req, &str)

	if len(errors) == 0 {
		t.Error("Expected validation errors for non-struct")
	}

	if errors["_struct"] == nil {
		t.Error("Expected struct validation errors")
	}
}

func TestCustomValidator(t *testing.T) {
	// Register a custom validator
	RegisterValidator("custom", func(value string) string {
		if value != "expected" {
			return "Value must be 'expected'"
		}
		return ""
	})

	form := url.Values{}
	form.Set("field", "wrong")

	req, _ := http.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	type CustomForm struct {
		Field string `form:"field" validate:"custom"`
	}

	var testForm CustomForm
	errors := DecodeAndValidate(req, &testForm)

	if len(errors) == 0 {
		t.Error("Expected validation errors for custom validator")
	}

	if errors["field"] == nil {
		t.Error("Expected field validation errors")
	}
}

func TestCustomSanitizer(t *testing.T) {
	// Register a custom sanitizer
	RegisterSanitizer("reverse", func(value string) string {
		runes := []rune(value)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes)
	})

	form := url.Values{}
	form.Set("field", "hello")

	req, _ := http.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	type CustomForm struct {
		Field string `form:"field" sanitize:"reverse"`
	}

	var testForm CustomForm
	errors := DecodeAndValidate(req, &testForm)

	if len(errors) > 0 {
		t.Errorf("Expected no validation errors, got: %v", errors)
	}

	if testForm.Field != "olleh" {
		t.Errorf("Expected field to be reversed 'olleh', got '%s'", testForm.Field)
	}
}

func TestValidationErrors_JSON(t *testing.T) {
	errors := ValidationErrors{
		"email":    []string{"Invalid email format"},
		"password": []string{"Must be at least 8 characters long"},
	}

	jsonData, err := json.Marshal(errors)
	if err != nil {
		t.Errorf("Failed to marshal validation errors to JSON: %v", err)
	}

	var unmarshaled ValidationErrors
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal validation errors from JSON: %v", err)
	}

	if len(unmarshaled) != len(errors) {
		t.Errorf("Expected %d error fields, got %d", len(errors), len(unmarshaled))
	}
}

func TestBuiltinValidators(t *testing.T) {
	testCases := []struct {
		name      string
		value     string
		validator string
		param     string
		expected  string
	}{
		{"required_empty", "", "required", "", "This field is required"},
		{"required_valid", "value", "required", "", ""},
		{"email_invalid", "invalid", "email", "", "Invalid email format"},
		{"email_valid", "test@example.com", "email", "", ""},
		{"min_short", "123", "min", "5", "Must be at least 5 characters long"},
		{"min_valid", "12345", "min", "5", ""},
		{"max_long", "123456", "max", "5", "Must be no more than 5 characters long"},
		{"max_valid", "12345", "max", "5", ""},
		{"url_invalid", "not-a-url", "url", "", "Invalid URL format"},
		{"url_valid", "https://example.com", "url", "", ""},
		{"numeric_invalid", "abc", "numeric", "", "Must be a number"},
		{"numeric_valid", "123", "numeric", "", ""},
		{"alpha_invalid", "abc123", "alpha", "", "Must contain only letters"},
		{"alpha_valid", "abc", "alpha", "", ""},
		{"alphanumeric_invalid", "abc@123", "alphanumeric", "", "Must contain only letters and numbers"},
		{"alphanumeric_valid", "abc123", "alphanumeric", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if validator, exists := builtinValidators[tc.validator]; exists {
				result := validator(tc.value, tc.param)
				if result != tc.expected {
					t.Errorf("Expected '%s', got '%s'", tc.expected, result)
				}
			} else {
				t.Errorf("Validator '%s' not found", tc.validator)
			}
		})
	}
}

func TestBuiltinSanitizers(t *testing.T) {
	testCases := []struct {
		name      string
		sanitizer string
		input     string
		expected  string
	}{
		{"trim", "trim", "  hello  ", "hello"},
		{"to_lower", "to_lower", "HELLO", "hello"},
		{"to_upper", "to_upper", "hello", "HELLO"},
		{"escape_html", "escape_html", "<script>alert('xss')</script>", "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"},
		{"strip_numeric", "strip_numeric", "abc123def", "abcdef"},
		{"strip_alpha", "strip_alpha", "abc123def", "123"},
		{"normalize_whitespace", "normalize_whitespace", "hello   world", "hello world"},
		{"remove_special_chars", "remove_special_chars", "hello@world#123", "hello world 123"},
		{"title_case", "title_case", "hello world", "Hello World"},
		{"camel_case", "camel_case", "hello world", "helloWorld"},
		{"snake_case", "snake_case", "Hello World", "hello_world"},
		{"kebab_case", "kebab_case", "Hello World", "hello-world"},
		{"remove_html_tags", "remove_html_tags", "<p>Hello <b>World</b></p>", "Hello World"},
		{"normalize_unicode", "normalize_unicode", "café", "café"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if sanitizer, exists := builtinSanitizers[tc.sanitizer]; exists {
				result := sanitizer(tc.input)
				if result != tc.expected {
					t.Errorf("Expected '%s', got '%s'", tc.expected, result)
				}
			} else {
				t.Errorf("Sanitizer '%s' not found", tc.sanitizer)
			}
		})
	}
}

func TestChainedSanitizers(t *testing.T) {
	// Test chaining multiple sanitizers
	input := "  Hello   World@123  "
	expected := "hello-world-123"

	result := applySanitizers(input, "trim,to_lower,kebab_case")
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestContextAwareValidators(t *testing.T) {
	// Register a context-aware validator for testing
	RegisterContextValidator("test_context", func(value, param string, ctx ValidationContext) string {
		if value == "invalid" && ctx.Get("other_field") == "trigger" {
			return "Context validation failed"
		}
		return ""
	})

	// Test context-aware validation
	context := ValidationContext{
		values: map[string]string{
			"test_field":  "invalid",
			"other_field": "trigger",
		},
	}

	validator := registry.contextValidators["test_context"]
	if validator == nil {
		t.Fatal("Context validator not registered")
	}

	result := validator("invalid", "", context)
	if result != "Context validation failed" {
		t.Errorf("Expected 'Context validation failed', got '%s'", result)
	}

	// Test with valid value
	result = validator("valid", "", context)
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

func TestDBLikeValidation(t *testing.T) {
	// Simulate a DB uniqueness check
	RegisterContextValidator("unique_email", func(value, param string, ctx ValidationContext) string {
		// Simulate taken emails
		takenEmails := map[string]bool{
			"admin@example.com": true,
			"test@example.com":  true,
		}

		if takenEmails[value] {
			return "Email is already registered"
		}
		return ""
	})

	// Test with taken email
	context := ValidationContext{values: make(map[string]string)}
	validator := registry.contextValidators["unique_email"]

	result := validator("admin@example.com", "", context)
	if result != "Email is already registered" {
		t.Errorf("Expected 'Email is already registered', got '%s'", result)
	}

	// Test with available email
	result = validator("new@example.com", "", context)
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

func TestCrossFieldValidation(t *testing.T) {
	// Test cross-field validation with context
	context := ValidationContext{
		values: map[string]string{
			"password":         "password123",
			"confirm_password": "password123",
			"start_date":       "2023-01-01",
			"end_date":         "2023-12-31",
		},
	}

	// Test eqfield validator
	eqfieldValidator := builtinContextValidators["eqfield"]
	result := eqfieldValidator("password123", "confirm_password", context)
	if result != "" {
		t.Errorf("Expected empty string for matching passwords, got '%s'", result)
	}

	// Test date_after validator
	dateAfterValidator := builtinContextValidators["date_after"]
	result = dateAfterValidator("2023-12-31", "start_date", context)
	if result != "" {
		t.Errorf("Expected empty string for valid date, got '%s'", result)
	}

	// Test invalid date
	result = dateAfterValidator("2022-12-31", "start_date", context)
	if result == "" {
		t.Error("Expected error for invalid date order")
	}
}

func TestSanitizationInValidation(t *testing.T) {
	// Test that sanitization is applied before validation
	form := url.Values{}
	form.Set("username", "  ADMIN  ")
	form.Set("email", "  TEST@EXAMPLE.COM  ")

	req, _ := http.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	type TestForm struct {
		Username string `form:"username" sanitize:"trim,to_lower" validate:"required,min=3"`
		Email    string `form:"email" sanitize:"trim,to_lower" validate:"required,email"`
	}

	var testForm TestForm
	errors := DecodeAndValidate(req, &testForm)

	if len(errors) > 0 {
		t.Errorf("Expected no validation errors, got: %v", errors)
	}

	// Check that sanitization was applied
	if testForm.Username != "admin" {
		t.Errorf("Expected username to be 'admin', got '%s'", testForm.Username)
	}

	if testForm.Email != "test@example.com" {
		t.Errorf("Expected email to be 'test@example.com', got '%s'", testForm.Email)
	}
}

func TestAdvancedConditionalValidation(t *testing.T) {
	tests := []struct {
		name     string
		form     interface{}
		request  *http.Request
		expected ValidationErrors
	}{
		{
			name: "required_if validation with field:value syntax",
			form: &struct {
				AccountType string `form:"account_type" validate:"required"`
				CompanyName string `form:"company_name" validate:"required_if=account_type:business"`
				TaxID       string `form:"tax_id" validate:"required_if=account_type:business"`
			}{},
			request: createRequest(map[string]string{
				"account_type": "business",
				"company_name": "",
				"tax_id":       "",
			}),
			expected: ValidationErrors{
				"company_name": []string{"This field is required when account_type is business"},
				"tax_id":       []string{"This field is required when account_type is business"},
			},
		},
		{
			name: "required_if validation with field only",
			form: &struct {
				Email   string `form:"email" validate:"required"`
				Phone   string `form:"phone" validate:"required_if=email"`
				Address string `form:"address"`
			}{},
			request: createRequest(map[string]string{
				"email":   "test@example.com",
				"phone":   "",
				"address": "123 Main St",
			}),
			expected: ValidationErrors{
				"phone": []string{"Invalid parameter for required_if"},
			},
		},
		{
			name: "required_unless validation",
			form: &struct {
				AccountType string `form:"account_type" validate:"required"`
				CompanyName string `form:"company_name" validate:"required_unless=account_type:personal"`
				TaxID       string `form:"tax_id" validate:"required_unless=account_type:personal"`
			}{},
			request: createRequest(map[string]string{
				"account_type": "business",
				"company_name": "",
				"tax_id":       "",
			}),
			expected: ValidationErrors{
				"company_name": []string{"This field is required"},
				"tax_id":       []string{"This field is required"},
			},
		},
		{
			name: "required_unless validation - exempt case",
			form: &struct {
				AccountType string `form:"account_type" validate:"required"`
				CompanyName string `form:"company_name" validate:"required_unless=account_type:personal"`
			}{},
			request: createRequest(map[string]string{
				"account_type": "personal",
				"company_name": "",
			}),
			expected: ValidationErrors{},
		},
		{
			name: "eqfield validation - matching",
			form: &struct {
				Password        string `form:"password" validate:"required"`
				ConfirmPassword string `form:"confirm_password" validate:"required,eqfield=password"`
			}{},
			request: createRequest(map[string]string{
				"password":         "secret123",
				"confirm_password": "secret123",
			}),
			expected: ValidationErrors{},
		},
		{
			name: "eqfield validation - not matching",
			form: &struct {
				Password        string `form:"password" validate:"required"`
				ConfirmPassword string `form:"confirm_password" validate:"required,eqfield=password"`
			}{},
			request: createRequest(map[string]string{
				"password":         "secret123",
				"confirm_password": "different123",
			}),
			expected: ValidationErrors{
				"confirm_password": []string{"Must match the \"password\" field"},
			},
		},
		{
			name: "nefield validation - different values",
			form: &struct {
				Username string `form:"username" validate:"required"`
				Email    string `form:"email" validate:"required,nefield=username"`
			}{},
			request: createRequest(map[string]string{
				"username": "john_doe",
				"email":    "john@example.com",
			}),
			expected: ValidationErrors{},
		},
		{
			name: "nefield validation - same values",
			form: &struct {
				Username string `form:"username" validate:"required"`
				Email    string `form:"email" validate:"required,nefield=username"`
			}{},
			request: createRequest(map[string]string{
				"username": "john_doe",
				"email":    "john_doe",
			}),
			expected: ValidationErrors{
				"email": []string{"Must not match the value of \"username\""},
			},
		},
		{
			name: "gtfield validation - greater",
			form: &struct {
				MinAge int `form:"min_age" validate:"required,numeric"`
				MaxAge int `form:"max_age" validate:"required,numeric,gtfield=min_age"`
			}{},
			request: createRequest(map[string]string{
				"min_age": "18",
				"max_age": "25",
			}),
			expected: ValidationErrors{},
		},
		{
			name: "gtfield validation - not greater",
			form: &struct {
				MinAge int `form:"min_age" validate:"required,numeric"`
				MaxAge int `form:"max_age" validate:"required,numeric,gtfield=min_age"`
			}{},
			request: createRequest(map[string]string{
				"min_age": "25",
				"max_age": "18",
			}),
			expected: ValidationErrors{
				"max_age": []string{"Must be greater than \"min_age\""},
			},
		},
		{
			name: "gtefield validation - greater than",
			form: &struct {
				MinAge int `form:"min_age" validate:"required,numeric"`
				MaxAge int `form:"max_age" validate:"required,numeric,gtefield=min_age"`
			}{},
			request: createRequest(map[string]string{
				"min_age": "18",
				"max_age": "25",
			}),
			expected: ValidationErrors{},
		},
		{
			name: "gtefield validation - equal",
			form: &struct {
				MinAge int `form:"min_age" validate:"required,numeric"`
				MaxAge int `form:"max_age" validate:"required,numeric,gtefield=min_age"`
			}{},
			request: createRequest(map[string]string{
				"min_age": "18",
				"max_age": "18",
			}),
			expected: ValidationErrors{},
		},
		{
			name: "gtefield validation - less than",
			form: &struct {
				MinAge int `form:"min_age" validate:"required,numeric"`
				MaxAge int `form:"max_age" validate:"required,numeric,gtefield=min_age"`
			}{},
			request: createRequest(map[string]string{
				"min_age": "25",
				"max_age": "18",
			}),
			expected: ValidationErrors{
				"max_age": []string{"Must be greater than or equal to \"min_age\""},
			},
		},
		{
			name: "ltfield validation - less",
			form: &struct {
				MaxPrice float64 `form:"max_price" validate:"required,numeric"`
				MinPrice float64 `form:"min_price" validate:"required,numeric,ltfield=max_price"`
			}{},
			request: createRequest(map[string]string{
				"max_price": "100.50",
				"min_price": "50.25",
			}),
			expected: ValidationErrors{},
		},
		{
			name: "ltfield validation - not less",
			form: &struct {
				MaxPrice float64 `form:"max_price" validate:"required,numeric"`
				MinPrice float64 `form:"min_price" validate:"required,numeric,ltfield=max_price"`
			}{},
			request: createRequest(map[string]string{
				"max_price": "50.25",
				"min_price": "100.50",
			}),
			expected: ValidationErrors{
				"min_price": []string{"Must be less than \"max_price\""},
			},
		},
		{
			name: "ltefield validation - less than",
			form: &struct {
				MaxPrice float64 `form:"max_price" validate:"required,numeric"`
				MinPrice float64 `form:"min_price" validate:"required,numeric,ltefield=max_price"`
			}{},
			request: createRequest(map[string]string{
				"max_price": "100.50",
				"min_price": "50.25",
			}),
			expected: ValidationErrors{},
		},
		{
			name: "ltefield validation - equal",
			form: &struct {
				MaxPrice float64 `form:"max_price" validate:"required,numeric"`
				MinPrice float64 `form:"min_price" validate:"required,numeric,ltefield=max_price"`
			}{},
			request: createRequest(map[string]string{
				"max_price": "100.50",
				"min_price": "100.50",
			}),
			expected: ValidationErrors{},
		},
		{
			name: "ltefield validation - greater than",
			form: &struct {
				MaxPrice float64 `form:"max_price" validate:"required,numeric"`
				MinPrice float64 `form:"min_price" validate:"required,numeric,ltefield=max_price"`
			}{},
			request: createRequest(map[string]string{
				"max_price": "50.25",
				"min_price": "100.50",
			}),
			expected: ValidationErrors{
				"min_price": []string{"Must be less than or equal to \"max_price\""},
			},
		},
		{
			name: "date_after validation - valid",
			form: &struct {
				StartDate string `form:"start_date" validate:"required"`
				EndDate   string `form:"end_date" validate:"required,date_after=start_date"`
			}{},
			request: createRequest(map[string]string{
				"start_date": "2024-01-01",
				"end_date":   "2024-12-31",
			}),
			expected: ValidationErrors{},
		},
		{
			name: "date_after validation - invalid",
			form: &struct {
				StartDate string `form:"start_date" validate:"required"`
				EndDate   string `form:"end_date" validate:"required,date_after=start_date"`
			}{},
			request: createRequest(map[string]string{
				"start_date": "2024-12-31",
				"end_date":   "2024-01-01",
			}),
			expected: ValidationErrors{
				"end_date": []string{"Must be after \"start_date\""},
			},
		},
		{
			name: "date_before validation - valid",
			form: &struct {
				EndDate   string `form:"end_date" validate:"required"`
				StartDate string `form:"start_date" validate:"required,date_before=end_date"`
			}{},
			request: createRequest(map[string]string{
				"end_date":   "2024-12-31",
				"start_date": "2024-01-01",
			}),
			expected: ValidationErrors{},
		},
		{
			name: "date_before validation - invalid",
			form: &struct {
				EndDate   string `form:"end_date" validate:"required"`
				StartDate string `form:"start_date" validate:"required,date_before=end_date"`
			}{},
			request: createRequest(map[string]string{
				"end_date":   "2024-01-01",
				"start_date": "2024-12-31",
			}),
			expected: ValidationErrors{
				"start_date": []string{"Must be before \"end_date\""},
			},
		},
		{
			name: "complex conditional validation",
			form: &struct {
				AccountType string `form:"account_type" validate:"required"`
				CompanyName string `form:"company_name" validate:"required_if=account_type:business"`
				TaxID       string `form:"tax_id" validate:"required_if=account_type:business"`
				StartYear   int    `form:"start_year" validate:"required,numeric"`
				EndYear     int    `form:"end_year" validate:"required,numeric,gtfield=start_year"`
				StartDate   string `form:"start_date" validate:"required"`
				EndDate     string `form:"end_date" validate:"required,date_after=start_date"`
			}{},
			request: createRequest(map[string]string{
				"account_type": "business",
				"company_name": "",
				"tax_id":       "",
				"start_year":   "2024",
				"end_year":     "2023",
				"start_date":   "2024-12-31",
				"end_date":     "2024-01-01",
			}),
			expected: ValidationErrors{
				"company_name": []string{"This field is required when account_type is business"},
				"tax_id":       []string{"This field is required when account_type is business"},
				"end_year":     []string{"Must be greater than \"start_year\""},
				"end_date":     []string{"Must be after \"start_date\""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := DecodeAndValidate(tt.request, tt.form)

			if len(errors) != len(tt.expected) {
				t.Errorf("Expected %d error fields, got %d", len(tt.expected), len(errors))
				t.Logf("Expected: %v", tt.expected)
				t.Logf("Got: %v", errors)
				return
			}

			for field, expectedErrors := range tt.expected {
				if actualErrors, exists := errors[field]; !exists {
					t.Errorf("Expected errors for field '%s', but none found", field)
				} else if len(actualErrors) != len(expectedErrors) {
					t.Errorf("Expected %d errors for field '%s', got %d", len(expectedErrors), field, len(actualErrors))
				} else {
					for i, expectedError := range expectedErrors {
						if i < len(actualErrors) && actualErrors[i] != expectedError {
							t.Errorf("Expected error '%s' for field '%s', got '%s'", expectedError, field, actualErrors[i])
						}
					}
				}
			}
		})
	}
}

func TestContextValidatorRegistration(t *testing.T) {
	// Test custom context validator registration
	customValidatorCalled := false
	RegisterContextValidator("custom_context", func(value, param string, context ValidationContext) string {
		customValidatorCalled = true
		if context.Get("other_field") == "expected_value" && value == "" {
			return "Custom context validation failed"
		}
		return ""
	})

	form := &struct {
		OtherField string `form:"other_field"`
		TestField  string `form:"test_field" validate:"custom_context"`
	}{}

	request := createRequest(map[string]string{
		"other_field": "expected_value",
		"test_field":  "",
	})

	errors := DecodeAndValidate(request, form)

	if !customValidatorCalled {
		t.Error("Custom context validator was not called")
	}

	if len(errors) == 0 || len(errors["test_field"]) == 0 {
		t.Error("Expected custom context validation to fail")
	}
}

func TestValidationContextGet(t *testing.T) {
	context := ValidationContext{
		values: map[string]string{
			"field1": "value1",
			"field2": "value2",
		},
	}

	if value := context.Get("field1"); value != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}

	if value := context.Get("field2"); value != "value2" {
		t.Errorf("Expected 'value2', got '%s'", value)
	}

	if value := context.Get("nonexistent"); value != "" {
		t.Errorf("Expected empty string for nonexistent field, got '%s'", value)
	}
}
