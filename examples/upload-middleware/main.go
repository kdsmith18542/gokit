package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kdsmith18542/gokit/upload"
	"github.com/kdsmith18542/gokit/upload/storage"
)

func main() {
	uploadDir := "./uploads"
	os.MkdirAll(uploadDir, 0755)
	localStorage := storage.NewLocal(uploadDir)
	processor := upload.NewProcessor(localStorage, upload.Options{
		MaxFileSize:      5 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg", "image/png", "application/pdf"},
	})

	mux := http.NewServeMux()
	mux.Handle("/upload", upload.UploadMiddleware(processor, "file", nil)(http.HandlerFunc(uploadHandler)))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))

	fmt.Println("upload middleware demo running at http://localhost:8083/upload")
	fmt.Println("Try:")
	fmt.Println("  - POST /upload (multipart: file)")
	fmt.Println("  - GET /uploads/<filename>")
	log.Fatal(http.ListenAndServe(":8083", mux))
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	results := upload.UploadResultsFromContext(r.Context())
	if results == nil || len(results) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "Uploaded %d file(s):\n", len(results))
	for _, result := range results {
		fmt.Fprintf(w, "- %s (%d bytes) -> /uploads/%s\n", result.OriginalName, result.Size, filepath.Base(result.Path))
	}
}
