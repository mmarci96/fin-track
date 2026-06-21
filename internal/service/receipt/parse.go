package receipt

import "context"

// minItems is the threshold below which a parse is flagged low_item_count (and,
// if a fallback is available, retried via the LLM).
const minItems = 2

// Parser turns OCR text into a structured Result. It is configured with the
// known canonical merchant names and an optional LLM fallback.
type Parser struct {
	known    []string
	fallback LLMFallback
}

// NewParser builds a parser. fallback may be nil (heuristics only), which is how
// the offline tests run.
func NewParser(known []string, fallback LLMFallback) *Parser {
	return &Parser{known: known, fallback: fallback}
}

// Parse runs the full pipeline and returns a graded Result.
func (p *Parser) Parse(ctx context.Context, text string) Result {
	lines := normalizeText(text)

	mm := detectMerchant(lines, p.known)
	res := Result{
		MerchantKnown: mm.Known,
		MerchantName:  mm.Candidate,
	}
	if mm.Known {
		res.MerchantName = mm.Canonical
	}

	ex := extractItems(lines)
	res.Items = ex.Items
	res.Total = ex.Total
	reconcile(&res)

	// LLM fallback: heuristics produced nothing usable or did not reconcile.
	if p.fallback != nil && p.shouldFallback(res) {
		if items, total, err := p.fallback.ExtractItems(ctx, text); err == nil {
			if better := p.adoptFallback(res, items, total); better != nil {
				res = *better
				res.Warnings = append(res.Warnings, WarnLLMFallbackUsed)
			}
		}
	}

	p.decide(&res)
	return res
}

// shouldFallback decides whether the LLM is worth invoking.
func (p *Parser) shouldFallback(r Result) bool {
	if len(r.Items) == 0 {
		return true
	}
	if r.Total > 0 && !r.Reconciled {
		return true
	}
	return len(r.Items) < minItems
}

// adoptFallback returns a new result built from the LLM output if it is a
// genuine improvement (more items, or reconciles when the heuristic did not).
// Merchant detection is never delegated to the LLM.
func (p *Parser) adoptFallback(base Result, items []Item, total int) *Result {
	if len(items) == 0 {
		return nil
	}

	candidate := base
	candidate.Items = items
	if total > 0 {
		candidate.Total = total
	}
	candidate.Warnings = nil
	reconcile(&candidate)

	switch {
	case candidate.Reconciled && !base.Reconciled:
		return &candidate
	case len(items) > len(base.Items):
		return &candidate
	default:
		return nil
	}
}

// decide applies the graded policy: zero items => reject (retake photo);
// otherwise accept, flagging anything unverified.
func (p *Parser) decide(r *Result) {
	if len(r.Items) == 0 {
		r.Decision = DecisionReject
		r.Confidence = 0
		return
	}

	if !r.MerchantKnown {
		r.Warnings = append(r.Warnings, WarnMerchantUnknown)
	}
	switch {
	case r.Total == 0:
		r.Warnings = append(r.Warnings, WarnTotalUnverified)
	case !r.Reconciled:
		r.Warnings = append(r.Warnings, WarnTotalMismatch)
	}
	if len(r.Items) < minItems {
		r.Warnings = append(r.Warnings, WarnLowItemCount)
	}

	if r.Reconciled && r.MerchantKnown {
		r.Decision = DecisionAccepted
	} else {
		r.Decision = DecisionBestEffort
	}
	r.Confidence = confidence(r)
}

// confidence is a rough 0..1 quality score for observability and thresholds.
func confidence(r *Result) float64 {
	score := 0.0
	if r.MerchantKnown {
		score += 0.25
	}
	if r.Total > 0 {
		score += 0.15
	}
	if r.Reconciled {
		score += 0.45
	}
	if len(r.Items) >= minItems {
		score += 0.15
	}
	return score
}
