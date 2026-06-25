package receipt

import (
	"strings"

	"github.com/agnivade/levenshtein"
)

// merchantThreshold is the minimum normalized similarity to accept a learned
// full-header alias as a confident match.
const merchantThreshold = 0.6

// brandThreshold is the minimum similarity between a header token and a known
// merchant's brand token. Higher than merchantThreshold because brands are short
// and distinctive, so OCR noise aside they should match closely.
const brandThreshold = 0.7

// strongThreshold is the minimum whole-string similarity to accept a match on the
// full normalized name. High enough that a shared corporate tail alone is not
// enough (AUCHAN…KFT vs ROSSMANN…KFT ≈ 0.72 is rejected) but a garbled-but-intact
// legal name still matches (ALUL… vs ALDI… ≈ 0.87).
const strongThreshold = 0.8

// headerScanLines is how many leading lines to scan for the merchant name. The
// brand can sit below an address block, so we look a little past the first line.
const headerScanLines = 6

// minBrandLen guards token matching: brands shorter than this must match exactly,
// since a 1-2 char Levenshtein edit on a tiny token is meaningless.
const minBrandLen = 3

// merchantStopwords are corporate/legal/geographic tokens that don't identify a
// brand. Stripping them stops shared tails like "MAGYARORSZAG KFT" from making
// distinct merchants (AUCHAN vs ROSSMANN) look similar.
var merchantStopwords = map[string]bool{
	"KFT": true, "ZRT": true, "BT": true, "NYRT": true, "RT": true, "KKT": true,
	"KERESKEDELMI": true, "ELELMISZER": true, "MAGYARORSZAG": true,
	"HUNGARY": true, "ES": true,
}

type merchantMatch struct {
	Canonical string  // canonical name from the known list ("" if unknown)
	Candidate string  // best-effort cleaned name from the receipt header
	Score     float64 // best matching score
	Known     bool
}

// detectMerchant matches the receipt header against the known canonical merchant
// names by their distinctive *brand* token, plus any learned full-header aliases.
// It never panics on empty lists and always returns a best-effort candidate so
// unknown merchants can still be recorded (normalized).
func detectMerchant(lines, known []string, aliases []MerchantAlias) merchantMatch {
	candidate := ""
	if len(lines) > 0 {
		candidate = strings.TrimSpace(lines[0])
	}

	best := merchantMatch{Candidate: candidate}

	// Precompute each known merchant's normalized name and brand (first
	// distinctive token).
	type knownMerchant struct{ norm, brand, canonical string }
	knownM := make([]knownMerchant, 0, len(known))
	for _, k := range known {
		brand := ""
		if toks := brandTokens(k); len(toks) > 0 {
			brand = toks[0]
		}
		knownM = append(knownM, knownMerchant{normalizeName(k), brand, k})
	}

	limit := min(headerScanLines, len(lines))
	for i := 0; i < limit; i++ {
		norm := normalizeName(lines[i])
		if norm == "" {
			continue
		}
		headerBrands := brandTokens(lines[i])

		for _, km := range knownM {
			// (a) brand-token match: catches clean/short headers and ignores the
			// shared corporate tail that made AUCHAN look like ROSSMANN.
			for _, tok := range headerBrands {
				if s := brandSimilarity(tok, km.brand); s >= brandThreshold && s > best.Score {
					best.Score, best.Canonical, best.Known = s, km.canonical, true
				}
			}
			// (b) whole-string match: catches a garbled-but-intact legal name where
			// no single token clears the brand bar (ALUL MAGYAROROZAG… → ALDI).
			if s := similarity(norm, km.norm); s >= strongThreshold && s > best.Score {
				best.Score, best.Canonical, best.Known = s, km.canonical, true
			}
		}

		// Alias match: learned full-header variants (long-to-long similarity), so a
		// garbled legal-name header resolves once it has been approved.
		for _, a := range aliases {
			if s := similarity(norm, a.Normalized); s >= merchantThreshold && s > best.Score {
				best.Score, best.Canonical, best.Known = s, a.Canonical, true
			}
		}
	}

	return best
}

// brandTokens returns the distinctive (non-stopword, non-numeric) tokens of a
// merchant/header name, in order.
func brandTokens(name string) []string {
	var out []string
	for _, tok := range strings.Fields(normalizeName(name)) {
		if merchantStopwords[tok] || isAllDigits(tok) {
			continue
		}
		out = append(out, tok)
	}
	return out
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// brandSimilarity compares two brand tokens, requiring an exact match for very
// short tokens where edit distance is not meaningful.
func brandSimilarity(a, b string) float64 {
	if len(a) < minBrandLen || len(b) < minBrandLen {
		if a == b {
			return 1
		}
		return 0
	}
	return similarity(a, b)
}

func similarity(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	dist := levenshtein.ComputeDistance(a, b)
	maxLen := max(len(a), len(b))
	return 1 - float64(dist)/float64(maxLen)
}

// strategyKey picks an extraction strategy from a (canonical or candidate)
// merchant name.
// func strategyKey(name string) string {
// 	n := normalizeName(name)
// 	switch {
// 	case strings.Contains(n, "ALDI"):
// 		return "aldi"
// 	case strings.Contains(n, "ROSSMANN"):
// 		return "rossmann"
// 	default:
// 		return "generic"
// 	}
// }
