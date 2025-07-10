package form

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/kdsmith18542/gokit/observability"
)

// Common form processing logic shared between form.go and json.go

// processFormFields processes form fields by collecting values, applying sanitizers, and setting field values
func processFormFields(val reflect.Value, formData map[string][]string) map[string]string {
	fieldValues := make(map[string]string)
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		formTag := fieldType.Tag.Get("form")
		if formTag == "" {
			formTag = strings.ToLower(fieldType.Name)
		}

		var value string
		if values := formData[formTag]; len(values) > 0 {
			value = values[0]
		}

		sanitizeTag := fieldType.Tag.Get("sanitize")
		if sanitizeTag != "" {
			value = applySanitizers(value, sanitizeTag)
		}

		fieldValues[formTag] = value
		// Also store by lowercase field name for cross-field validation
		fieldValues[strings.ToLower(fieldType.Name)] = value
		if field.CanSet() {
			setFieldValue(field, value)
		}
	}

	return fieldValues
}

// validateFormFields validates all form fields using the validation context
func validateFormFields(val reflect.Value, fieldValues map[string]string) ValidationErrors {
	errors := make(ValidationErrors)
	typ := val.Type()

	validationContext := ValidationContext{values: fieldValues}
	for i := 0; i < val.NumField(); i++ {
		fieldType := typ.Field(i)
		formTag := fieldType.Tag.Get("form")
		if formTag == "" {
			formTag = strings.ToLower(fieldType.Name)
		}
		value := fieldValues[formTag]
		validateTag := fieldType.Tag.Get("validate")
		if validateTag != "" {
			fieldErrors := validateFieldWithContext(value, validateTag, validationContext, fieldType.Type.Kind())
			if len(fieldErrors) > 0 {
				errors[formTag] = fieldErrors
			}
		}
	}

	return errors
}

// validateStructPointer validates that the target is a non-nil pointer to struct
func validateStructPointer(ctx context.Context, v interface{}, formName string) ValidationErrors {
	errors := make(ValidationErrors)

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

	return nil
}

// handleFormObservability handles observability for form processing
func handleFormObservability(ctx context.Context, formName string, errors ValidationErrors, start time.Time) {
	duration := time.Since(start)
	if obs := getObserver(); obs != nil {
		obs.OnValidationEnd(ctx, formName, errors)
	}

	// Update the formObserver to include duration
	if _, ok := observer.(*formObserver); ok {
		observability.GetObserver().OnFormValidationEnd(ctx, formName, len(errors), duration)
	}
}
