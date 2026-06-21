package receipt

import (
	"strings"

	"github.com/agnivade/levenshtein"
)

// merchantThreshold is the minimum normalized similarity to consider a known
// merchant a confident match.
const merchantThreshold = 0.6

// headerScanLines is how many leading lines to scan for the merchant name.
const headerScanLines = 3

type merchantMatch struct {
	Canonical string  // canonical name from the known list ("" if unknown)
	Candidate string  // best-effort cleaned name from the receipt header
	Score     float64 // best normalized similarity
	Known     bool
}

// detectMerchant matches the receipt header against the known canonical
// merchant names. It never panics on an empty known list and always returns a
// best-effort candidate so unknown merchants can still be recorded (normalized).
func detectMerchant(lines, known []string) merchantMatch {
	candidate := ""
	if len(lines) > 0 {
		candidate = strings.TrimSpace(lines[0])
	}

	best := merchantMatch{Candidate: candidate}

	limit := min(headerScanLines, len(lines))
	for i := 0; i < limit; i++ {
		norm := normalizeName(lines[i])
		if norm == "" {
			continue
		}
		for _, k := range known {
			s := similarity(norm, normalizeName(k))
			if s > best.Score {
				best.Score = s
				if s >= merchantThreshold {
					best.Canonical = k
					best.Known = true
				}
			}
		}
	}

	return best
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
func strategyKey(name string) string {
	n := normalizeName(name)
	switch {
	case strings.Contains(n, "ALDI"):
		return "aldi"
	case strings.Contains(n, "ROSSMANN"):
		return "rossmann"
	default:
		return "generic"
	}
}
