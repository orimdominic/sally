package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
	"github.com/orimdominic/sally/server/internal/genkitai"
	bgtask "github.com/orimdominic/sally/server/internal/task"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		err = bgtask.StartServer()
		if err != nil {
			log.Fatal(err)
		}
	}()

	ctx := context.Background()
	gktMngr, err := genkitai.NewGenkitManager(ctx)
	if err != nil {
		log.Fatal(err)
	}

	redisOpt := asynq.RedisClientOpt{Addr: "localhost:6379"}
	bgTasksClient := asynq.NewClient(redisOpt)
	err = bgTasksClient.Ping()
	if err != nil {
		log.Fatal(err)
	}
	defer bgTasksClient.Close()

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	fs := http.FileServer(http.Dir("./static"))
	r.Handle("/*", http.StripPrefix("/", fs))
	r.Post("/documents", handleFileUpload(bgTasksClient))
	r.Get("/query", handleQuery(gktMngr))
	r.Get("/publickey", handleGetPublicKey)

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

	log.Println("Shutting down...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}
}

type pushSubscription struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func handleFileUpload(bgTaskClient *asynq.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const maxUploadSize = 10 << 20 // 10 * 1024 * 1024 i.e 10 MB
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
		if err != nil || !ok {
			fmt.Println(err)
			http.Error(w, "Invalid pdf file", http.StatusBadRequest)
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

		subVal := r.FormValue("subscription")
		var subscription pushSubscription
		err = json.Unmarshal([]byte(subVal), &subscription)
		if err != nil {
			fmt.Println("Invalid subscription key")
			http.Error(w, "Invalid subscription key. Retry", http.StatusInternalServerError)
			return
		}

		t, err := bgtask.NewIndexPDFDocTask(dstPath, &webpush.Subscription{
			Endpoint: subscription.Endpoint,
			Keys: webpush.Keys{
				Auth:   subscription.Keys.Auth,
				P256dh: subscription.Keys.P256dh,
			},
		})
		if err != nil {
			fmt.Printf("could not create %s task. %v\n", bgtask.TypeIndexPDFDoc, err)
			http.Error(w, "Failed to index document. Retry", http.StatusInternalServerError)
			return
		}

		_, err = bgTaskClient.Enqueue(t)
		if err != nil {
			fmt.Printf("could not start document processing. %v\n", err)
			http.Error(w, "Failed to index document. Retry", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(
			w,
			"%s saved. You will be notified when processing is complete",
			fileHeader.Filename,
		)
	}
}

type queryResults struct {
	Results []string `json:"results"`
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

		res := queryResults{
			Results: results,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&res)
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

type publicKeyResponse struct {
	PublicKey string `json:"publicKey"`
}

func handleGetPublicKey(w http.ResponseWriter, _ *http.Request) {
	VAPID_PUBLIC_KEY := os.Getenv("VAPID_PUBLIC_KEY")
	json.NewEncoder(w).Encode(&publicKeyResponse{
		PublicKey: VAPID_PUBLIC_KEY,
	})
}
