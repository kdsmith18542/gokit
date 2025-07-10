package i18n

import (
	"context"
	"net/http"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

const (
	// TranslatorContextKey is the key used to store the Translator in the request context
	TranslatorContextKey contextKey = "gokit_translator"
	// LocaleContextKey is the key used to store the detected locale in the request context
	LocaleContextKey contextKey = "gokit_locale"
)

// LocaleDetector returns middleware that detects the user's locale and injects
// a pre-configured Translator into the request context.
//
// This middleware runs once per request and stores the Translator in the context,
// eliminating the need for repeated locale detection in handlers. It automatically
// detects the locale from query parameters, cookies, or Accept-Language headers.
//
// Example usage:
//
//	func main() {
//	    // Initialize i18n manager
//	    manager := i18n.NewManager("./locales")
//
//	    mux := http.NewServeMux()
//	    mux.HandleFunc("/", homeHandler)
//	    mux.HandleFunc("/products", productsHandler)
//	    mux.HandleFunc("/about", aboutHandler)
//
//	    // Apply locale detection middleware to all routes
//	    handler := i18n.LocaleDetector(manager)(mux)
//	    http.ListenAndServe(":8080", handler)
//	}
//
//	func homeHandler(w http.ResponseWriter, r *http.Request) {
//	    translator := i18n.TranslatorFromContext(r.Context())
//
//	    // Use translator for all text
//	    data := map[string]interface{}{
//	        "title": translator.T("home.title"),
//	        "welcome": translator.T("home.welcome"),
//	        "stats": translator.T("home.stats", map[string]interface{}{
//	            "users": 1234,
//	            "products": 567,
//	        }),
//	        "lastUpdated": translator.FormatDate(time.Now()),
//	    }
//
//	    renderTemplate(w, "home.html", data)
//	}
//
//	func productsHandler(w http.ResponseWriter, r *http.Request) {
//	    translator := i18n.TranslatorFromContext(r.Context())
//
//	    products := []Product{
//	        {Name: "Product 1", Price: 29.99},
//	        {Name: "Product 2", Price: 49.99},
//	    }
//
//	    // Format prices according to locale
//	    for i := range products {
//	        products[i].FormattedPrice = translator.FormatCurrency(products[i].Price, "USD")
//	    }
//
//	    data := map[string]interface{}{
//	        "title": translator.T("products.title"),
//	        "products": products,
//	        "total": translator.FormatCurrency(calculateTotal(products), "USD"),
//	    }
//
//	    renderTemplate(w, "products.html", data)
//	}
//
// Locale detection order:
// 1. Query parameter: ?locale=es
// 2. Cookie: locale=es
// 3. Accept-Language header: Accept-Language: es-ES,es;q=0.9,en;q=0.8
// 4. Default locale (configured in manager)
//
// The middleware supports:
// - Automatic locale detection from multiple sources
// - Fallback to default locale if detection fails
// - Context injection for easy access in handlers
// - Performance optimization (detection runs once per request)
func LocaleDetector(manager *Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Detect locale and create translator
			translator := manager.Translator(r)

			// Store translator and locale in context
			ctx := context.WithValue(r.Context(), TranslatorContextKey, translator)
			ctx = context.WithValue(ctx, LocaleContextKey, translator.locale.Code)

			// Create new request with updated context
			r = r.WithContext(ctx)

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// TranslatorFromContext retrieves the Translator from the request context.
// Returns nil if no Translator was found in the context.
//
// This function should be used in handlers when the LocaleDetector middleware
// has been applied to the request.
//
// Example:
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    translator := i18n.TranslatorFromContext(r.Context())
//	    if translator == nil {
//	        // Fallback to default locale or handle error
//	        http.Error(w, "No translator available", http.StatusInternalServerError)
//	        return
//	    }
//
//	    // Use translator for internationalization
//	    greeting := translator.T("welcome_message")
//	    userCount := translator.T("user_count", map[string]interface{}{
//	        "count": 42,
//	    })
//	    formattedDate := translator.FormatDate(time.Now())
//
//	    // Send response
//	    w.Header().Set("Content-Type", "application/json")
//	    json.NewEncoder(w).Encode(map[string]interface{}{
//	        "greeting": greeting,
//	        "user_count": userCount,
//	        "date": formattedDate,
//	    })
//	}
//
// For guaranteed access (when you're certain the middleware is applied):
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    translator := i18n.MustTranslatorFromContext(r.Context())
//
//	    // translator is guaranteed to be non-nil
//	    message := translator.T("some_key")
//	}
func TranslatorFromContext(ctx context.Context) *Translator {
	if translator, ok := ctx.Value(TranslatorContextKey).(*Translator); ok {
		return translator
	}
	return nil
}

// LocaleFromContext retrieves the detected locale code from the request context.
// Returns an empty string if no locale was found in the context.
//
// Example:
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    locale := i18n.LocaleFromContext(r.Context())
//
//	    // Use locale for conditional logic
//	    if locale == "es" {
//	        // Spanish-specific logic
//	        w.Header().Set("Content-Language", "es")
//	    } else if locale == "fr" {
//	        // French-specific logic
//	        w.Header().Set("Content-Language", "fr")
//	    }
//
//	    // Or use it for logging/analytics
//	    log.Printf("Request from locale: %s", locale)
//	}
//
// You can also use it for template rendering:
//
//	func renderTemplate(w http.ResponseWriter, templateName string, data interface{}) {
//	    locale := i18n.LocaleFromContext(r.Context())
//
//	    // Load locale-specific template
//	    templatePath := fmt.Sprintf("templates/%s/%s.html", locale, templateName)
//	    tmpl, err := template.ParseFiles(templatePath)
//	    if err != nil {
//	        // Fallback to default template
//	        tmpl, _ = template.ParseFiles(fmt.Sprintf("templates/en/%s.html", templateName))
//	    }
//
//	    tmpl.Execute(w, data)
//	}
func LocaleFromContext(ctx context.Context) string {
	if locale, ok := ctx.Value(LocaleContextKey).(string); ok {
		return locale
	}
	return ""
}

// MustTranslatorFromContext retrieves the Translator from the request context.
// Panics if no Translator was found in the context.
//
// Use this function when you're certain the LocaleDetector middleware
// has been applied and you want to fail fast if it hasn't.
func MustTranslatorFromContext(ctx context.Context) *Translator {
	translator := TranslatorFromContext(ctx)
	if translator == nil {
		panic("i18n: Translator not found in context. Did you apply the LocaleDetector middleware?")
	}
	return translator
}

// LocaleDetectorWithFallback returns middleware that detects the user's locale
// and injects a Translator into the request context, with a fallback translator
// if detection fails.
//
// This is useful when you want to ensure a translator is always available,
// even if locale detection fails. It's particularly helpful for applications
// that need to handle edge cases where locale detection might not work.
//
// Example usage:
//
//	func main() {
//	    manager := i18n.NewManager("./locales")
//
//	    mux := http.NewServeMux()
//	    mux.HandleFunc("/", homeHandler)
//
//	    // Use fallback middleware to ensure translator is always available
//	    handler := i18n.LocaleDetectorWithFallback(manager, "en")(mux)
//	    http.ListenAndServe(":8080", handler)
//	}
//
//	func homeHandler(w http.ResponseWriter, r *http.Request) {
//	    // Translator is guaranteed to be available (falls back to "en")
//	    translator := i18n.MustTranslatorFromContext(r.Context())
//
//	    // Safe to use without nil checks
//	    greeting := translator.T("welcome_message")
//	    locale := i18n.LocaleFromContext(r.Context())
//
//	    log.Printf("Serving request in locale: %s", locale)
//
//	    // Process request...
//	}
//
// This middleware is recommended when:
// - You need guaranteed translator availability
// - Handling requests from unknown locales
// - Building robust applications that shouldn't fail on locale issues
// - Supporting fallback scenarios gracefully
func LocaleDetectorWithFallback(manager *Manager, fallbackLocale string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Detect locale and create translator
			translator := manager.Translator(r)

			// If no translator was found, create one with fallback locale
			if translator == nil || translator.locale == nil {
				// Create a fallback translator
				fallbackTranslator := &Translator{
					locale:  manager.getLocale(fallbackLocale),
					manager: manager,
				}
				translator = fallbackTranslator
			}

			// Store translator and locale in context
			ctx := context.WithValue(r.Context(), TranslatorContextKey, translator)
			ctx = context.WithValue(ctx, LocaleContextKey, translator.locale.Code)

			// Create new request with updated context
			r = r.WithContext(ctx)

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// LocaleDetectorWithOptions returns middleware with configurable options.
type LocaleDetectorOptions struct {
	// FallbackLocale is the locale to use if detection fails
	FallbackLocale string
	// SetCookie sets a cookie with the detected locale
	SetCookie bool
	// CookieName is the name of the cookie to set (default: "locale")
	CookieName string
	// CookieMaxAge is the max age of the locale cookie in seconds (default: 1 year)
	CookieMaxAge int
}

// LocaleDetectorWithOptions returns middleware with configurable options.
//
// This is the most flexible middleware that allows fine-grained control over
// locale detection behavior, including cookie management and fallback handling.
//
// Example usage:
//
//	func main() {
//	    manager := i18n.NewManager("./locales")
//
//	    mux := http.NewServeMux()
//	    mux.HandleFunc("/", homeHandler)
//	    mux.HandleFunc("/api/users", apiHandler)
//
//	    // Configure middleware with options
//	    options := i18n.LocaleDetectorOptions{
//	        FallbackLocale: "en",
//	        SetCookie:      true,
//	        CookieName:     "app_locale",
//	        CookieMaxAge:   30 * 24 * 60 * 60, // 30 days
//	    }
//
//	    handler := i18n.LocaleDetectorWithOptions(manager, options)(mux)
//	    http.ListenAndServe(":8080", handler)
//	}
//
//	func homeHandler(w http.ResponseWriter, r *http.Request) {
//	    translator := i18n.MustTranslatorFromContext(r.Context())
//	    locale := i18n.LocaleFromContext(r.Context())
//
//	    // The cookie will be automatically set by the middleware
//	    // for future requests from this user
//
//	    data := map[string]interface{}{
//	        "title": translator.T("home.title"),
//	        "locale": locale,
//	    }
//
//	    renderTemplate(w, "home.html", data)
//	}
//
//	func apiHandler(w http.ResponseWriter, r *http.Request) {
//	    translator := i18n.MustTranslatorFromContext(r.Context())
//
//	    // API responses with localized messages
//	    w.Header().Set("Content-Type", "application/json")
//	    json.NewEncoder(w).Encode(map[string]interface{}{
//	        "message": translator.T("api.success"),
//	        "locale":  i18n.LocaleFromContext(r.Context()),
//	    })
//	}
//
// Cookie behavior:
// - When SetCookie is true, the detected locale is stored in a cookie
// - The cookie is accessible to JavaScript (HttpOnly: false)
// - Future requests will use the cookie value for locale detection
// - This provides persistent locale preference for users
//
// Use cases for different options:
// - SetCookie: true - For user preference persistence
// - FallbackLocale: "en" - For robust fallback handling
// - Custom CookieName - For multi-tenant applications
// - Custom CookieMaxAge - For different session lengths
func LocaleDetectorWithOptions(manager *Manager, opts LocaleDetectorOptions) func(http.Handler) http.Handler {
	// Set defaults
	if opts.CookieName == "" {
		opts.CookieName = "locale"
	}
	if opts.CookieMaxAge == 0 {
		opts.CookieMaxAge = 365 * 24 * 60 * 60 // 1 year
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Detect locale and create translator
			translator := manager.Translator(r)

			// If no translator was found and fallback is specified, use fallback
			if (translator == nil || translator.locale == nil) && opts.FallbackLocale != "" {
				fallbackTranslator := &Translator{
					locale:  manager.getLocale(opts.FallbackLocale),
					manager: manager,
				}
				translator = fallbackTranslator
			}

			// Set cookie if requested and we have a valid locale
			if opts.SetCookie && translator != nil && translator.locale != nil {
				http.SetCookie(w, &http.Cookie{
					Name:     opts.CookieName,
					Value:    translator.locale.Code,
					MaxAge:   opts.CookieMaxAge,
					Path:     "/",
					HttpOnly: false, // Allow JavaScript access for client-side locale switching
				})
			}

			// Store translator and locale in context
			ctx := context.WithValue(r.Context(), TranslatorContextKey, translator)
			ctx = context.WithValue(ctx, LocaleContextKey, translator.locale.Code)

			// Create new request with updated context
			r = r.WithContext(ctx)

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}
