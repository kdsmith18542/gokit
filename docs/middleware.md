# GoKit HTTP Middleware Guide

GoKit provides **optional, idiomatic HTTP middleware** for i18n, form validation, and file upload. Middleware is not required—it's a convenience for context-based handler design. You can always use the lower-level APIs directly for full control.

## Philosophy
- **Optional**: Middleware is never required. All core APIs work without it.
- **Composable**: Works with any `net/http`-compatible router or framework.
- **Idiomatic**: Encourages context-based handler design, reducing boilerplate.
- **Non-intrusive**: Does not enforce application structure or routing.

---

## i18n Middleware

### Usage
```go
mux := http.NewServeMux()
mux.HandleFunc("/greet", greetHandler)
handler := i18n.LocaleDetector(i18nManager)(mux)
http.ListenAndServe(":8080", handler)
```

### In Your Handler
```go
func greetHandler(w http.ResponseWriter, r *http.Request) {
    translator := i18n.TranslatorFromContext(r.Context())
    greeting := translator.T("welcome", map[string]interface{}{"Name": "User"})
    fmt.Fprint(w, greeting)
}
```

### Options
- `LocaleDetectorWithFallback(manager, fallbackLocale)`
- `LocaleDetectorWithOptions(manager, opts)`
- Extract locale code: `i18n.LocaleFromContext(r.Context())`

---

## Form Validation Middleware

### Usage
```go
type UserForm struct {
    Email    string `form:"email" validate:"required,email"`
    Password string `form:"password" validate:"required,min=8"`
}
mux.Handle("/register", form.ValidationMiddleware(UserForm{}, nil)(registerHandler))
```

### In Your Handler
```go
func registerHandler(w http.ResponseWriter, r *http.Request) {
    form := form.ValidatedFormFromContext(r.Context()).(*UserForm)
    // Use validated form
}
```

### Error Handling
- Default: JSON error response
- Custom: Pass your own error handler to `ValidationMiddleware`

---

## Upload Middleware

### Usage
```go
mux.Handle("/upload", upload.UploadMiddleware(processor, "file", nil)(uploadHandler))
```

### In Your Handler
```go
func uploadHandler(w http.ResponseWriter, r *http.Request) {
    results := upload.UploadResultsFromContext(r.Context())
    // Use upload results
}
```

### Error Handling
- Default: JSON error response
- Custom: Pass your own error handler to `UploadMiddleware`

---

## Combining Middleware

You can combine GoKit middleware with each other and with other middleware:

```go
mux := http.NewServeMux()
mux.Handle("/register", form.ValidationMiddleware(UserForm{}, nil)(http.HandlerFunc(registerHandler)))
mux.Handle("/upload", upload.UploadMiddleware(processor, "file", nil)(http.HandlerFunc(uploadHandler)))
handler := i18n.LocaleDetector(i18nManager)(mux)
http.ListenAndServe(":8080", handler)
```

---

## Integration Tips
- Works with any `net/http` router (e.g., chi, gorilla/mux, echo, gin with adapters)
- Middleware order matters: wrap your router or specific routes as needed
- You can mix GoKit middleware with your own or third-party middleware

---

## FAQ

**Q: Do I have to use middleware?**
A: No. All APIs work without it. Middleware is just a convenience.

**Q: Can I use GoKit middleware with my favorite router?**
A: Yes! GoKit middleware is standard `func(http.Handler) http.Handler` and works with any router that supports `net/http` handlers.

**Q: How do I access the translator, form, or upload results?**
A: Use the provided context helpers in your handler:
- `i18n.TranslatorFromContext(r.Context())`
- `form.ValidatedFormFromContext(r.Context())`
- `upload.UploadResultsFromContext(r.Context())`
- `upload.SingleUploadResultFromContext(r.Context())` (for single file uploads)

**Q: Can I customize error handling?**
A: Yes. Pass your own error handler to the middleware constructor.

---

For more, see the README and the `examples/` directory. 