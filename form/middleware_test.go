package form

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type TestFormMiddleware struct {
	Email    string `form:"email" validate:"required,email"`
	Password string `form:"password" validate:"required,min=8"`
	Name     string `form:"name" sanitize:"trim"`
}

func TestValidationMiddleware(t *testing.T) {
	// Test valid form
	t.Run("valid form", func(t *testing.T) {
		form := TestFormMiddleware{}
		middleware := ValidationMiddleware(form, nil)

		// Create form data
		data := url.Values{}
		data.Set("email", "test@example.com")
		data.Set("password", "password123")
		data.Set("name", "  John Doe  ")

		req := httptest.NewRequest("POST", "/", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		var capturedForm interface{}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedForm = ValidatedFormFromContext(r.Context())
		})

		middleware(handler).ServeHTTP(w, req)

		// Check response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Check form was captured and sanitized
		if capturedForm == nil {
			t.Error("Form not found in context")
		} else {
			form := capturedForm.(*TestFormMiddleware)
			if form.Email != "test@example.com" {
				t.Errorf("Expected email 'test@example.com', got '%s'", form.Email)
			}
			if form.Password != "password123" {
				t.Errorf("Expected password 'password123', got '%s'", form.Password)
			}
			if form.Name != "John Doe" { // Should be trimmed
				t.Errorf("Expected name 'John Doe', got '%s'", form.Name)
			}
		}
	})

	// Test invalid form
	t.Run("invalid form", func(t *testing.T) {
		form := TestFormMiddleware{}
		middleware := ValidationMiddleware(form, nil)

		// Create invalid form data
		data := url.Values{}
		data.Set("email", "invalid-email")
		data.Set("password", "short")

		req := httptest.NewRequest("POST", "/", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called for invalid form")
		})

		middleware(handler).ServeHTTP(w, req)

		// Check response
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		// Check JSON response
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}

		if response["error"] != "Validation failed" {
			t.Errorf("Expected error 'Validation failed', got '%v'", response["error"])
		}

		details, ok := response["details"].(map[string]interface{})
		if !ok {
			t.Error("Expected details in response")
		}

		// Check for email error
		if emailErrors, ok := details["email"].([]interface{}); !ok || len(emailErrors) == 0 {
			t.Error("Expected email validation error")
		}

		// Check for password error
		if passwordErrors, ok := details["password"].([]interface{}); !ok || len(passwordErrors) == 0 {
			t.Error("Expected password validation error")
		}
	})
}

func TestValidationMiddlewareWithContext(t *testing.T) {
	form := TestFormMiddleware{}
	middleware := ValidationMiddlewareWithContext(form, nil)

	// Create form data
	data := url.Values{}
	data.Set("email", "test@example.com")
	data.Set("password", "password123")

	req := httptest.NewRequest("POST", "/", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	var capturedForm interface{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedForm = ValidatedFormFromContext(r.Context())
	})

	middleware(handler).ServeHTTP(w, req)

	// Check form was captured
	if capturedForm == nil {
		t.Error("Form not found in context")
	}
}

func TestValidatedFormFromContext(t *testing.T) {
	// Test with nil context
	form := ValidatedFormFromContext(context.Background())
	if form != nil {
		t.Error("Expected nil form for empty context")
	}

	// Test with context containing form
	testForm := &TestFormMiddleware{Email: "test@example.com"}
	ctx := context.WithValue(context.Background(), formDataKey, testForm)

	form = ValidatedFormFromContext(ctx)
	if form != testForm {
		t.Error("Expected form from context")
	}
}

func TestMustValidatedFormFromContext(t *testing.T) {
	// Test panic case
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when form not found in context")
		}
	}()

	MustValidatedFormFromContext(context.Background())
}

func TestMustValidatedFormFromContextSuccess(t *testing.T) {
	// Test success case
	testForm := &TestFormMiddleware{Email: "test@example.com"}
	ctx := context.WithValue(context.Background(), formDataKey, testForm)

	form := MustValidatedFormFromContext(ctx)
	if form != testForm {
		t.Error("Expected form from context")
	}
}

func TestJSONValidationErrorHandler(t *testing.T) {
	errors := ValidationErrors{
		"email":    []string{"Invalid email format"},
		"password": []string{"Password too short"},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", nil)

	JSONValidationErrorHandler(w, req, errors)

	// Check response
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d", w.Code)
	}

	// Check JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "error" {
		t.Errorf("Expected status 'error', got '%v'", response["status"])
	}

	if response["message"] != "Validation failed" {
		t.Errorf("Expected message 'Validation failed', got '%v'", response["message"])
	}

	// Check errors array
	errorList, ok := response["errors"].([]interface{})
	if !ok {
		t.Error("Expected errors array in response")
	}

	if len(errorList) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errorList))
	}
}

func TestHTMLValidationErrorHandler(t *testing.T) {
	errors := ValidationErrors{
		"email":    []string{"Invalid email format"},
		"password": []string{"Password too short"},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", nil)

	HTMLValidationErrorHandler(w, req, errors)

	// Check response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Validation Error") {
		t.Error("Expected 'Validation Error' in HTML response")
	}

	if !strings.Contains(body, "Invalid email format") {
		t.Error("Expected email error in HTML response")
	}

	if !strings.Contains(body, "Password too short") {
		t.Error("Expected password error in HTML response")
	}
}

func TestCustomValidationErrorHandler(t *testing.T) {
	customHandler := func(w http.ResponseWriter, r *http.Request, errs ValidationErrors) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("Custom error handler"))
	}

	form := TestFormMiddleware{}
	middleware := ValidationMiddleware(form, customHandler)

	// Create invalid form data
	data := url.Values{}
	data.Set("email", "invalid-email")

	req := httptest.NewRequest("POST", "/", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for invalid form")
	})

	middleware(handler).ServeHTTP(w, req)

	// Check custom response
	if w.Code != http.StatusTeapot {
		t.Errorf("Expected status 418, got %d", w.Code)
	}

	if w.Body.String() != "Custom error handler" {
		t.Errorf("Expected 'Custom error handler', got '%s'", w.Body.String())
	}
}
