package form

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
)

// DecodeAndValidateJSON decodes JSON data from an io.Reader and validates it against a struct.
// This function supports the same validation and sanitization features as DecodeAndValidate.
//
// Example:
//
//	var user User
//	errors := form.DecodeAndValidateJSON(ctx, r.Body, &user)
//	if len(errors) > 0 {
//	    // Handle validation errors
//	}
func DecodeAndValidateJSON(ctx context.Context, reader io.Reader, v interface{}) ValidationErrors {
	start := time.Now()
	formName := ""
	if v != nil {
		formName = reflect.TypeOf(v).Elem().Name()
	}

	if obs := getObserver(); obs != nil {
		obs.OnDecodeStart(ctx, formName)
	}

	errors := make(ValidationErrors)

	// Decode JSON into a map
	var jsonData map[string]interface{}
	if err := json.NewDecoder(reader).Decode(&jsonData); err != nil {
		errors["_json"] = []string{"Failed to decode JSON: " + err.Error()}
		if obs := getObserver(); obs != nil {
			obs.OnDecodeEnd(ctx, formName, err)
		}
		return errors
	}

	// Convert map to form-like structure
	formData := make(map[string][]string)
	for key, value := range jsonData {
		formData[key] = []string{toString(value)}
	}

	// Validate struct
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

	// First pass: collect all field values and apply sanitizers
	fieldValues := processFormFields(val, formData)

	if obs := getObserver(); obs != nil {
		obs.OnDecodeEnd(ctx, formName, nil)
		obs.OnValidationStart(ctx, formName)
	}

	// Second pass: validate fields
	errors = validateFormFields(val, fieldValues)

	handleFormObservability(ctx, formName, errors, start)

	return errors
}

// DecodeAndValidateMap decodes and validates data from a map[string]interface{}.
// This is useful for programmatic validation or when working with parsed JSON data.
//
// Example:
//
//	data := map[string]interface{}{
//	    "email": "user@example.com",
//	    "age":   25,
//	}
//	var user User
//	errors := form.DecodeAndValidateMap(ctx, data, &user)
func DecodeAndValidateMap(ctx context.Context, data map[string]interface{}, v interface{}) ValidationErrors {
	start := time.Now()
	formName := ""
	if v != nil {
		formName = reflect.TypeOf(v).Elem().Name()
	}

	if obs := getObserver(); obs != nil {
		obs.OnDecodeStart(ctx, formName)
	}

	errors := make(ValidationErrors)

	// Convert map to form-like structure
	formData := make(map[string][]string)
	for key, value := range data {
		formData[key] = []string{toString(value)}
	}

	// Validate struct
	if structErrors := validateStructPointer(v, ctx, formName); structErrors != nil {
		return structErrors
	}
	val := reflect.ValueOf(v).Elem()

	// First pass: collect all field values and apply sanitizers
	fieldValues := processFormFields(val, formData)

	if obs := getObserver(); obs != nil {
		obs.OnDecodeEnd(ctx, formName, nil)
		obs.OnValidationStart(ctx, formName)
	}

	// Second pass: validate fields
	errors = validateFormFields(val, fieldValues)

	handleFormObservability(ctx, formName, errors, start)

	return errors
}

// toString converts any value to a string representation.
// This handles various JSON types (string, number, boolean, null).
func toString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case float64:
		// JSON numbers are always float64
		if v == float64(int(v)) {
			return fmt.Sprintf("%.0f", v)
		}
		return fmt.Sprintf("%g", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case []interface{}:
		// For arrays, join with comma
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = toString(item)
		}
		return strings.Join(parts, ",")
	case map[string]interface{}:
		// For objects, convert to JSON string
		if jsonBytes, err := json.Marshal(v); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
