package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/kdsmith18542/gokit/i18n"
)

func main() {
	i18nManager := i18n.NewManager("../locales")
	mux := http.NewServeMux()
	mux.HandleFunc("/greet", greetHandler)

	handler := i18n.LocaleDetector(i18nManager)(mux)

	fmt.Println("i18n middleware demo running at http://localhost:8081/greet")
	fmt.Println("Try:")
	fmt.Println("  - GET /greet?locale=es or Accept-Language: es")
	log.Fatal(http.ListenAndServe(":8081", handler))
}

func greetHandler(w http.ResponseWriter, r *http.Request) {
	translator := i18n.TranslatorFromContext(r.Context())
	if translator == nil {
		translator = i18n.NewManager("../locales").Translator(r)
	}
	greeting := translator.T("welcome", map[string]interface{}{"Name": "Demo User"})
	fmt.Fprint(w, greeting)
}
