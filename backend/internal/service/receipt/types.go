// Package receipt turns raw OCR text into a structured, validated receipt. It
// is deliberately independent of OCR, HTTP, and the database so it can be unit
// tested offline against golden text fixtures.
//
// Pipeline: normalize -> detect merchant -> classify lines -> extract items ->
// reconcile against the printed total -> decide (accept / best-effort / reject)
// with an optional LLM fallback when heuristics are not confident.
package receipt

import "context"

// Item is a single line item on the receipt.
type Item struct {
	Name  string `json:"name"`
	Price int    `json:"price"`
}

// Decision is the graded outcome of parsing, driving the HTTP response.
type Decision string

const (
	// DecisionAccepted: items found and the total reconciles.
	DecisionAccepted Decision = "accepted"
	// DecisionBestEffort: items found but unverified (no/garbled total, unknown
	// merchant, or total did not reconcile). Still stored, but flagged.
	DecisionBestEffort Decision = "best_effort"
	// DecisionReject: nothing usable parsed. The client should retake the photo.
	DecisionReject Decision = "reject"
)

// Warning codes attached to a best-effort result.
const (
	WarnMerchantUnknown = "merchant_unknown"
	WarnTotalUnverified = "total_unverified"
	WarnTotalMismatch   = "total_mismatch"
	WarnLowItemCount    = "low_item_count"
	WarnLLMFallbackUsed = "llm_fallback_used"
)

// Result is the structured output of parsing.
type Result struct {
	MerchantName  string   `json:"merchant_name"`
	MerchantKnown bool     `json:"merchant_known"`
	Items         []Item   `json:"items"`
	Total         int      `json:"total"`          // printed total, 0 if not found
	ComputedTotal int      `json:"computed_total"` // sum of item prices
	Reconciled    bool     `json:"reconciled"`
	Confidence    float64  `json:"confidence"`
	Decision      Decision `json:"decision"`
	Warnings      []string `json:"warnings,omitempty"`
}

// LLMFallback extracts items from OCR text when heuristics are not confident.
// Implemented by an Ollama-backed adapter in the service layer; the parser only
// depends on this interface so it stays testable offline.
type LLMFallback interface {
	ExtractItems(ctx context.Context, text string) ([]Item, int, error)
}
