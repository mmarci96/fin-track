package ollama

import (
	"context"
	"encoding/json"

	"github.com/cockroachdb/errors"
	"github.com/mmarci96/fin-track/internal/model"
	"github.com/mmarci96/fin-track/internal/service/receipt"
)

// ItemExtractor adapts the Ollama service to receipt.LLMFallback: it asks the
// model to pull structured line items out of noisy OCR text. It is only invoked
// when the heuristic parser is not confident.
type ItemExtractor struct {
	svc   *Service
	model string
}

func NewItemExtractor(svc *Service, model string) *ItemExtractor {
	return &ItemExtractor{svc: svc, model: model}
}

const extractPrompt = `You extract line items from a Hungarian shop receipt that was OCR-scanned (so text is noisy).
Return ONLY JSON of the form {"items":[{"name":"...","price":0}],"total":0}.
Rules:
- price and total are integers in HUF, no spaces or currency symbols.
- include only purchased products; EXCLUDE totals (OSSZEG/OSSZESEN), discounts (ENGEDMENY), deposits (visszavalt), card/payment lines, tax id and barcodes.
- if you cannot read an item, skip it. Do not invent items.

Receipt text:
`

type extractResponse struct {
	Items []receipt.Item `json:"items"`
	Total int            `json:"total"`
}

// ExtractItems implements receipt.LLMFallback.
func (e *ItemExtractor) ExtractItems(ctx context.Context, text string) ([]receipt.Item, int, error) {
	resp, err := e.svc.Generate(ctx, model.GenerateRequest{
		Model:  e.model,
		Prompt: extractPrompt + text,
		Stream: false,
		Format: "json",
	})
	if err != nil {
		return nil, 0, errors.Wrap(err, "ollama extract items")
	}

	var out extractResponse
	if err := json.Unmarshal([]byte(resp.Response), &out); err != nil {
		return nil, 0, errors.Wrapf(err, "decode llm json: %q", resp.Response)
	}

	return out.Items, out.Total, nil
}
