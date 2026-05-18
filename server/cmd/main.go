package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	// "strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/orimdominic/sally/server/internal/genkitai"
)

const maxUploadSize = 10 << 20 // 10 * 1024 * 1024 i.e 10 MB

func main() {
	ctx := context.Background()
	gktMngr, err := genkitai.NewGenkit(ctx)
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post("/documents", handleFileUpload(gktMngr))
	r.Post("/query", handleQuery(gktMngr))

	port := ":8888"
	srv := &http.Server{
		Addr:    port,
		Handler: r,
	}
	go func() {
		fmt.Printf("Server running on %s\n", port)
		if err := srv.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	<-sigCh

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	log.Println("shutting down...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}
}

func handleFileUpload(gktMngr *genkitai.GenkitManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		ok, err := IsPDF(file)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Invalid file", http.StatusBadRequest)
			return
		}

		if !ok {
			fmt.Println(err)
			http.Error(w, "Only .pdf is acceptable", http.StatusBadRequest)
			return
		}

		uploadDir := "./uploads"
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			fmt.Println(err)
			http.Error(w, "Unable to create dir", http.StatusInternalServerError)
			return
		}

		dstPath := filepath.Join(uploadDir, fileHeader.Filename)
		dst, err := os.Create(dstPath)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Failed to create destination file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		_, err = io.Copy(dst, file)
		if err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		err = gktMngr.IndexPDFDocument(r.Context(), dstPath)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Failed to index document", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s saved", fileHeader.Filename)
	}
}

func handleQuery(gktMngr *genkitai.GenkitManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.FormValue("query")

		results, err := gktMngr.QueryDocument(r.Context(), query)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Unable to get results", http.StatusInternalServerError)
			return
		}

		var sb strings.Builder
		for _, s := range results {
			sb.Write([]byte(s))
			sb.Write([]byte("\n\n"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sb.String()))
	}
}

func IsPDF(file multipart.File) (bool, error) {
	buffer := make([]byte, 512)

	n, err := file.Read(buffer)
	if err != nil {
		return false, err
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return false, err
	}

	contentType := http.DetectContentType(buffer[:n])

	return contentType == "application/pdf", nil
}
