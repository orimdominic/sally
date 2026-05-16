package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/orimdominic/sally/server/internal/embedder"
)

const maxUploadSize = 10 << 20 // 10 * 1024 * 1024 i.e 10 MB

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post("/documents", handleFileUpload)

	port := ":8888"
	fmt.Printf("Server starting on %s\n", port)
	http.ListenAndServe(port, r)
}

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File not found", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if filepath.Ext(fileHeader.Filename) != ".pdf" {
		fmt.Println(err)
		http.Error(w, "Only .pdf is acceptable", http.StatusBadRequest)
		return
	}

	filename := filepath.Base(fileHeader.Filename)
	destPath := filepath.Join("./uploads", filename)
	dest, err := os.Create(destPath)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
	defer dest.Close()

	b, err := io.ReadAll(file)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Unable to read file contents", http.StatusInternalServerError)
		return
	}

	_, err = embedder.Embed(string(b))
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error generating embeddings", http.StatusBadGateway)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s saved", filename)
}
