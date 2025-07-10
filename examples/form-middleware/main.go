package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/kdsmith18542/gokit/form"
)

type UserForm struct {
	Email    string `form:"email" validate:"required,email"`
	Password string `form:"password" validate:"required,min=8"`
	Name     string `form:"name" validate:"required"`
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/register", form.ValidationMiddleware(UserForm{}, nil)(http.HandlerFunc(registerHandler)))

	fmt.Println("form validation middleware demo running at http://localhost:8082/register")
	fmt.Println("Try:")
	fmt.Println("  - POST /register (form: email, password, name)")
	log.Fatal(http.ListenAndServe(":8082", mux))
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	formVal := form.ValidatedFormFromContext(r.Context())
	userForm, ok := formVal.(*UserForm)
	if !ok {
		http.Error(w, "Form not found in context", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "Registration successful for %s (%s)\n", userForm.Name, userForm.Email)
}
