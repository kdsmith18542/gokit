package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

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

	server := &http.Server{
		Addr:         ":8081",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}

func greetHandler(w http.ResponseWriter, r *http.Request) {
	translator := i18n.TranslatorFromContext(r.Context())
	if translator == nil {
		translator = i18n.NewManager("../locales").Translator(r)
	}
	greeting := translator.T("welcome", map[string]interface{}{"Name": "Demo User"})
	fmt.Fprint(w, greeting)
}
