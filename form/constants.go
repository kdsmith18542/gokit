package form

// Common validation error messages
const (
	ErrFieldRequired      = "This field is required"
	ErrMustBeNumber       = "Must be a number"
	ErrInvalidEmail       = "Invalid email format"
	ErrInvalidURL         = "Invalid URL format"
	ErrMustBeAlpha        = "Must contain only letters"
	ErrMustBeAlphanumeric = "Must contain only letters and numbers"
)

// Common test values
const (
	TestEmail     = "test@example.com"
	TestFile      = "test.txt"
	TestValue     = "test"
	TestJPG       = "test.jpg"
	TestBefore    = "before"
	TestHola      = "Hola"
	TestCompleted = "completed"
)
