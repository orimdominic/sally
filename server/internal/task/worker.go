package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/orimdominic/sally/server/internal/genkitai"
	"github.com/orimdominic/sally/server/internal/pushnotif"
)

func StartServer() error {
	redisOpt := asynq.RedisClientOpt{Addr: "localhost:6379"}

	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{Concurrency: 10},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeIndexPDFDoc, handleIndexPDF)

	if err := srv.Run(mux); err != nil {
		fmt.Printf("could not start redis server: %v", err)
		return err
	}

	return nil
}

func handleIndexPDF(ctx context.Context, t *asynq.Task) error {
	var p IndexPDFDocPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	gktMngr, err := genkitai.NewGenkitManager(ctx)
	if err != nil {
		return err
	}

	err = gktMngr.IndexPDFDocument(ctx, p.FPath)
	if err != nil {
		return pushnotif.NotifyEmbeddingFailed(p.ClientSub)
	}

	return pushnotif.NotifyEmbeddingCompleted(p.ClientSub)
}
