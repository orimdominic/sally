package task

import (
	"encoding/json"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/hibiken/asynq"
)

const TypeIndexPDFDoc = "index:pdf"

type IndexPDFDocPayload struct {
	// FPath is the file path
	FPath     string
	ClientSub *webpush.Subscription
}

func NewIndexPDFDocTask(
	fpath string,
	clientSubscription *webpush.Subscription,
) (*asynq.Task, error) {
	payload, err := json.Marshal(IndexPDFDocPayload{
		FPath:     fpath,
		ClientSub: clientSubscription,
	})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeIndexPDFDoc, payload), nil
}
