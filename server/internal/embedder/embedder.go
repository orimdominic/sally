package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/firebase/genkit/go/ai"
)

type RemoteEmbedder struct {
	URL string
}

type RemoteEmbeddingRequest struct {
	Texts []string `json:"texts"`
}

type RemoteEmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func generateRemoteEmbedder(url string) ai.EmbedderFunc {
	return func(ctx context.Context,
		req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
		texts := make([]string, len(req.Input))
		for i, d := range req.Input {
			texts[i] = d.Content[0].Text
		}

		body, _ := json.Marshal(RemoteEmbeddingRequest{
			Texts: texts,
		})

		httpReq, _ := http.NewRequestWithContext(
			ctx,
			"POST",
			url,
			bytes.NewBuffer(body),
		)

		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var out RemoteEmbeddingResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, err
		}

		embeddings := make([]*ai.Embedding, len(out.Embeddings))

		for i, e := range out.Embeddings {
			embeddings[i] = &ai.Embedding{
				Embedding: e,
			}
		}

		return &ai.EmbedResponse{
			Embeddings: embeddings,
		}, nil
	}
}

func NewRemoteEmbedder(url string) ai.Embedder {
	return ai.NewEmbedder(
		"HuggingFace-Xenova/all-MiniLM-L6-v2", &ai.EmbedderOptions{
			Dimensions: 384,
			Supports: &ai.EmbedderSupports{
				Input:        []string{"text"},
				Multilingual: false,
			},
		}, generateRemoteEmbedder(url))
}
