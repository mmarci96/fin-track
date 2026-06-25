package receipt

import "strings"

// Receipts print the payable amount several ways and OCR garbles individual
// lines, so rather than trust the first total-looking line we gather every
// candidate and pick the one closest to the reconciliation target (item sum
// adjusted for deposits/discounts). A garbled "BANKKARTYA: 0 Jol Et" is dropped;
// a clean "OSSZESEN 5 360" wins.

type candGroup int

const (
	groupGrand    candGroup = iota // OSSZESEN / FIZETENDO / VEGOSSZEG
	groupSubtotal                  // RESZOSSZESEN
	groupPayment                   // BANKKARTYA / KESZPENZ
)

type totalCandidate struct {
	value int
	group candGroup
}

// paymentKeywords mark a tendered/paid amount, a useful total source when the
// grand-total line is OCR-garbled.
var paymentKeywords = []string{"BANKKARTYA", "KESZPENZ", "KARTYA"}

// collectTotalCandidates scans all lines for total-, subtotal-, and payment-type
// amounts, parsing each value with the same column-aware lastPrice the item
// extractor uses. Non-positive / unparseable values are skipped.
func collectTotalCandidates(lines []string) []totalCandidate {
	var cands []totalCandidate
	for _, line := range lines {
		price, _, ok := lastPrice(line)
		if !ok || price <= 0 {
			continue
		}
		norm := normalizeName(line)
		switch {
		case strings.Contains(norm, "RESZOSSZESEN") || strings.Contains(norm, "RESZOSSZ"):
			cands = append(cands, totalCandidate{price, groupSubtotal})
		case containsAny(norm, totalKeywords):
			cands = append(cands, totalCandidate{price, groupGrand})
		case containsAny(norm, paymentKeywords):
			cands = append(cands, totalCandidate{price, groupPayment})
		}
	}
	return cands
}

// chooseTotal returns the candidate closest to target. If the closest reconciles
// (within tolerance) it wins; otherwise a grand-total line is preferred for
// display, falling back to the closest. Returns 0 when there are no candidates.
func chooseTotal(cands []totalCandidate, target int) int {
	if len(cands) == 0 {
		return 0
	}
	best := cands[0]
	for _, c := range cands[1:] {
		if abs(c.value-target) < abs(best.value-target) {
			best = c
		}
	}
	if target > 0 && abs(best.value-target) <= reconcileTolerance(target) {
		return best.value
	}
	for _, c := range cands {
		if c.group == groupGrand {
			return c.value
		}
	}
	return best.value
}
