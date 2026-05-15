package embedder

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type requestJson struct {
	Content string `json:"content"`
}

type responseJson struct {
	Embeddings []float64 `json:"embeddings"`
}

func Embed(content string) ([]float64, error) {
	b, err := json.Marshal(requestJson{Content: content})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(
		"http://localhost:3333",
		"application/json",
		bytes.NewBuffer(b),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data responseJson
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	return data.Embeddings, nil
}
