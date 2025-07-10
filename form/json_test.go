package form

import (
	"context"
	"strings"
	"testing"
)

// UserRegistrationForm is used for testing JSON validation
type UserRegistrationForm struct {
	Email           string `form:"email" validate:"required,email"`
	Password        string `form:"password" validate:"required,min=8"`
	ConfirmPassword string `form:"confirm_password" validate:"required,eqfield=Password"`
	Name            string `form:"name" validate:"required" sanitize:"trim"`
	Age             int    `form:"age" validate:"required,min=18"`
}

func TestDecodeAndValidateJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		form     interface{}
		wantErr  bool
		errCount int
	}{
		{
			name:     "valid json",
			jsonData: `{"email":"user@example.com","password":"password123","confirm_password":"password123","name":"John Doe","age":25}`,
			form:     &UserRegistrationForm{},
			wantErr:  false,
			errCount: 0,
		},
		{
			name:     "invalid email",
			jsonData: `{"email":"invalid-email","password":"password123","confirm_password":"password123","name":"John Doe","age":25}`,
			form:     &UserRegistrationForm{},
			wantErr:  true,
			errCount: 1,
		},
		{
			name:     "password mismatch",
			jsonData: `{"email":"user@example.com","password":"password123","confirm_password":"different","name":"John Doe","age":25}`,
			form:     &UserRegistrationForm{},
			wantErr:  true,
			errCount: 1,
		},
		{
			name:     "missing required fields",
			jsonData: `{"email":"user@example.com"}`,
			form:     &UserRegistrationForm{},
			wantErr:  true,
			errCount: 4, // password, confirm_password, name, age
		},
		{
			name:     "invalid json",
			jsonData: `{"email":"user@example.com",}`,
			form:     &UserRegistrationForm{},
			wantErr:  true,
			errCount: 1, // JSON decode error
		},
		{
			name:     "nil target",
			jsonData: `{"email":"user@example.com"}`,
			form:     nil,
			wantErr:  true,
			errCount: 1,
		},
		{
			name:     "non-struct target",
			jsonData: `{"email":"user@example.com"}`,
			form:     &[]string{},
			wantErr:  true,
			errCount: 1,
		},
		{
			name:     "json with numbers",
			jsonData: `{"email":"user@example.com","password":"password123","confirm_password":"password123","name":"John Doe","age":25.0}`,
			form:     &UserRegistrationForm{},
			wantErr:  false,
			errCount: 0,
		},
		{
			name:     "json with booleans",
			jsonData: `{"email":"user@example.com","password":"password123","confirm_password":"password123","name":"John Doe","age":25,"active":true}`,
			form:     &UserRegistrationForm{},
			wantErr:  false,
			errCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			reader := strings.NewReader(tt.jsonData)

			errors := DecodeAndValidateJSON(ctx, reader, tt.form)

			if tt.wantErr {
				if len(errors) == 0 {
					t.Errorf("DecodeAndValidateJSON() expected errors but got none")
				}
				if len(errors) != tt.errCount {
					t.Errorf("DecodeAndValidateJSON() error count = %d, want %d", len(errors), tt.errCount)
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("DecodeAndValidateJSON() unexpected errors: %v", errors)
				}
			}
		})
	}
}

func TestDecodeAndValidateMap(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		form     interface{}
		wantErr  bool
		errCount int
	}{
		{
			name: "valid map",
			data: map[string]interface{}{
				"email":            "user@example.com",
				"password":         "password123",
				"confirm_password": "password123",
				"name":             "John Doe",
				"age":              25,
			},
			form:     &UserRegistrationForm{},
			wantErr:  false,
			errCount: 0,
		},
		{
			name: "invalid email",
			data: map[string]interface{}{
				"email":            "invalid-email",
				"password":         "password123",
				"confirm_password": "password123",
				"name":             "John Doe",
				"age":              25,
			},
			form:     &UserRegistrationForm{},
			wantErr:  true,
			errCount: 1,
		},
		{
			name: "missing fields",
			data: map[string]interface{}{
				"email": "user@example.com",
			},
			form:     &UserRegistrationForm{},
			wantErr:  true,
			errCount: 4,
		},
		{
			name: "nil target",
			data: map[string]interface{}{
				"email": "user@example.com",
			},
			form:     nil,
			wantErr:  true,
			errCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			errors := DecodeAndValidateMap(ctx, tt.data, tt.form)

			if tt.wantErr {
				if len(errors) == 0 {
					t.Errorf("DecodeAndValidateMap() expected errors but got none")
				}
				if len(errors) != tt.errCount {
					t.Errorf("DecodeAndValidateMap() error count = %d, want %d", len(errors), tt.errCount)
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("DecodeAndValidateMap() unexpected errors: %v", errors)
				}
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string", "hello", "hello"},
		{"nil", nil, ""},
		{"int float", float64(42), "42"},
		{"float", float64(3.14), "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"array", []interface{}{"a", "b", "c"}, "a,b,c"},
		{"object", map[string]interface{}{"key": "value"}, `{"key":"value"}`},
		{"complex object", map[string]interface{}{"nested": map[string]interface{}{"key": "value"}}, `{"nested":{"key":"value"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toString(tt.input)
			if result != tt.expected {
				t.Errorf("toString(%v) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestJSONWithSanitization(t *testing.T) {
	jsonData := `{"name":"  John Doe  ","email":"USER@EXAMPLE.COM"}`

	type TestForm struct {
		Name  string `form:"name" sanitize:"trim"`
		Email string `form:"email" sanitize:"to_lower"`
	}

	var form TestForm
	ctx := context.Background()
	reader := strings.NewReader(jsonData)

	errors := DecodeAndValidateJSON(ctx, reader, &form)

	if len(errors) > 0 {
		t.Errorf("DecodeAndValidateJSON() unexpected errors: %v", errors)
	}

	if form.Name != "John Doe" {
		t.Errorf("Expected sanitized name 'John Doe', got '%s'", form.Name)
	}

	if form.Email != "user@example.com" {
		t.Errorf("Expected sanitized email 'user@example.com', got '%s'", form.Email)
	}
}

func TestJSONWithValidation(t *testing.T) {
	jsonData := `{"email":"user@example.com","password":"short","confirm_password":"short","name":"John","age":15}`

	var form UserRegistrationForm
	ctx := context.Background()
	reader := strings.NewReader(jsonData)

	errors := DecodeAndValidateJSON(ctx, reader, &form)

	if len(errors) == 0 {
		t.Error("Expected validation errors but got none")
	}

	// Check specific validation errors
	if _, hasEmail := errors["email"]; hasEmail {
		t.Error("Email should be valid")
	}

	if _, hasPassword := errors["password"]; !hasPassword {
		t.Error("Password should have validation error (too short)")
	}

	if _, hasAge := errors["age"]; !hasAge {
		t.Error("Age should have validation error (too young)")
	}
}

func TestJSONPerformance(t *testing.T) {
	jsonData := `{"email":"user@example.com","password":"password123","confirm_password":"password123","name":"John Doe","age":25}`

	ctx := context.Background()

	// Run a few validations to test basic performance
	for i := 0; i < 10; i++ {
		var form UserRegistrationForm // Create new instance each time
		reader := strings.NewReader(jsonData)
		errors := DecodeAndValidateJSON(ctx, reader, &form)
		if len(errors) > 0 {
			t.Errorf("Unexpected errors on iteration %d: %v", i, errors)
		}

		// Verify the form was populated correctly
		if form.Email != "user@example.com" {
			t.Errorf("Expected email 'user@example.com', got '%s'", form.Email)
		}
		if form.Age != 25 {
			t.Errorf("Expected age 25, got %d", form.Age)
		}
	}
}
