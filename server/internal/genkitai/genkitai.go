package genkitai

import (
	"bytes"
	"context"

	"github.com/dslipak/pdf"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/localvec"
	"github.com/orimdominic/sally/server/internal/embedder"
	"github.com/tmc/langchaingo/textsplitter"
)

type GenkitManager struct {
	gkt       *genkit.Genkit
	splitter  textsplitter.RecursiveCharacter
	docStore  *localvec.DocStore
	retriever ai.Retriever
}

var indexPDFFlow *core.Flow[string, any, struct{}]
var queryDocFlow *core.Flow[string, []string, struct{}]

func (mngr *GenkitManager) IndexPDFDocument(ctx context.Context, path string) error {
	_, err := indexPDFFlow.Run(ctx, path)
	if err != nil {
		return err
	}

	return nil
}

func (mngr *GenkitManager) QueryDocument(
	ctx context.Context, query string,
) ([]string, error) {
	return queryDocFlow.Run(ctx, query)
}

func (mngr *GenkitManager) RegisterFlows() {
	indexPDFFlow = genkit.DefineFlow(
		mngr.gkt,
		"index_pdf",
		func(ctx context.Context, path string) (any, error) {
			pdfText, err := genkit.Run(ctx, "extract", func() (string, error) {
				return readPdf(path)
			})
			if err != nil {
				return nil, err
			}

			docs, err := genkit.Run(ctx, "chunk", func() ([]*ai.Document, error) {
				chunks, err := mngr.splitter.SplitText(pdfText)
				if err != nil {
					return nil, err
				}

				var docs []*ai.Document
				for _, chunk := range chunks {
					docs = append(docs, ai.DocumentFromText(chunk, nil))
				}

				return docs, nil
			})
			if err != nil {
				return nil, err
			}

			// Add the chunks to the index using the vector store
			if err := localvec.Index(ctx, docs, mngr.docStore); err != nil {
				return nil, err
			}

			return map[string]any{
				"success":          true,
				"documentsIndexed": len(docs),
			}, nil
		})

	queryDocFlow = genkit.DefineFlow(
		mngr.gkt, "query",
		func(ctx context.Context, question string) ([]string, error) {
			// Retrieve text relevant to the user's question.
			var results []string
			resp, err := genkit.Retrieve(
				ctx,
				mngr.gkt,
				ai.WithRetriever(mngr.retriever),
				ai.WithTextDocs(question),
				ai.WithConfig(&localvec.RetrieverOptions{
					K: 5,
				}),
			)
			if err != nil {
				return results, err
			}

			for _, doc := range resp.Documents {
				for _, part := range doc.Content {
					results = append(results, part.Text)
				}
			}
			return results, nil
		})
}

func NewGenkit(ctx context.Context) (*GenkitManager, error) {
	var gkt *genkit.Genkit
	if ctx != nil {
		gkt = genkit.Init(ctx)
	} else {
		gkt = genkit.Init(context.Background())
	}

	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(200),
		textsplitter.WithChunkOverlap(20),
	)

	if err := localvec.Init(); err != nil {
		return nil, err
	}

	docStore, retriever, err := localvec.DefineRetriever(
		gkt,
		"document",
		localvec.Config{
			Dir:      "./embeddings",
			Embedder: embedder.NewRemoteEmbedder("http://localhost:3333"),
		},
		nil,
	)

	if err != nil {
		return nil, err
	}

	mngr := &GenkitManager{
		gkt:       gkt,
		splitter:  splitter,
		docStore:  docStore,
		retriever: retriever,
	}

	mngr.RegisterFlows()
	return mngr, nil
}

// readPdf is a helper function to extract plain text from a PDF. Excerpted from
func readPdf(docPath string) (string, error) {
	r, err := pdf.Open(docPath)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		return "", err
	}
	buf.ReadFrom(b)

	return buf.String(), nil
}
