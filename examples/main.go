// Package main provides example usage of the gokit form, i18n, and upload packages.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kdsmith18542/gokit/form"
	"github.com/kdsmith18542/gokit/i18n"
	"github.com/kdsmith18542/gokit/upload"
	"github.com/kdsmith18542/gokit/upload/storage"
)

// UserForm represents a user registration form.
type UserForm struct {
	Email    string `form:"email" validate:"required,email"`
	Password string `form:"password" validate:"required,min=8"`
	Name     string `form:"name" validate:"required"`
}

func main() {
	// Set up i18n manager with example locales
	i18nManager := i18n.NewManager("./examples/locales")

	// Set up upload processor (local storage)
	uploadDir := "./uploads"
	os.MkdirAll(uploadDir, 0755)
	localStorage := storage.NewLocal(uploadDir)
	fileProcessor := upload.NewProcessor(localStorage, upload.Options{
		MaxFileSize:      5 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg", "image/png", "application/pdf"},
		MaxFiles:         5,
	})

	mux := http.NewServeMux()

	// Registration form with validation middleware
	mux.Handle("/register", form.ValidationMiddleware(UserForm{}, nil)(http.HandlerFunc(registerHandler)))

	// File upload with upload middleware
	mux.Handle("/upload", upload.Middleware(fileProcessor, "file", nil)(http.HandlerFunc(uploadHandler)))

	// Greeting endpoint with i18n
	mux.HandleFunc("/greet", greetHandler)

	// Serve uploaded files for demo
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))

	// Wrap the entire mux with i18n.LocaleDetector middleware
	handler := i18n.LocaleDetector(i18nManager)(mux)

	fmt.Println("Demo server running at http://localhost:8080/")
	fmt.Println("Try:")
	fmt.Println("  - POST /register (form: email, password, name)")
	fmt.Println("  - POST /upload (multipart: file)")
	fmt.Println("  - GET /greet?locale=es or Accept-Language: es")
	fmt.Println("  - GET /uploads/<filename>")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

// Handler for registration form
func registerHandler(w http.ResponseWriter, r *http.Request) {
	// Extract validated form from context
	formVal := form.ValidatedFormFromContext(r.Context())
	userForm, ok := formVal.(*UserForm)
	if !ok {
		http.Error(w, "Form not found in context", http.StatusInternalServerError)
		return
	}

	// Get translator from context
	translator := i18n.TranslatorFromContext(r.Context())
	greeting := "Welcome!"
	if translator != nil {
		greeting = translator.T("welcome", map[string]interface{}{"Name": userForm.Name})
	}

	fmt.Fprintf(w, "%s Registration successful for %s\n", greeting, userForm.Email)
}

// Handler for file upload
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Extract upload results from context
	results := upload.ResultsFromContext(r.Context())
	if results == nil || len(results) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "Uploaded %d file(s):\n", len(results))
	for _, result := range results {
		fmt.Fprintf(w, "- %s (%d bytes) -> /uploads/%s\n", result.OriginalName, result.Size, filepath.Base(result.Path))
	}
}

// Handler for greeting with i18n
func greetHandler(w http.ResponseWriter, r *http.Request) {
	translator := i18n.TranslatorFromContext(r.Context())
	if translator == nil {
		translator = i18n.NewManager("./examples/locales").Translator(r)
	}
	greeting := translator.T("welcome", map[string]interface{}{"Name": "Visitor"})
	fmt.Fprint(w, greeting)
}
