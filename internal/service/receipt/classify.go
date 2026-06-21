package receipt

import (
	"regexp"
	"strings"
	"unicode"
)

type lineType int

const (
	lineNoise lineType = iota
	lineItem
	lineName // a name with no price; may pair with a following weight line
	lineWeight
	lineDiscount
	lineDeposit
	lineTotal
	lineFooter
)

var (
	longNumberRe = regexp.MustCompile(`\d{8,}`)
	ftPerKgRe    = regexp.MustCompile(`(?i)FT\s*/\s*KG`)
)

var (
	totalKeywords   = []string{"OSSZESEN", "OSSZEG", "FIZETENDO", "VEGOSSZEG"}
	depositKeywords = []string{"VISSZAVALT", "BETETDIJ"}
	discountKeyword = []string{"ENGEDMENY", "AKCIO", "KEDVEZMENY"}
	footerKeywords  = []string{
		"BANKKARTYA", "KESZPENZ", "VISA", "MASTERCARD", "CONTACTLESS",
		"AUTH", "ADOSZAM", "TERMINAL", "AID", "APP", "EAN", "FAN",
	}
)

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func countLetters(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			n++
		}
	}
	return n
}

// classify assigns a semantic type to a cleaned line. hasPrice reports whether
// the line carries a trailing price token, which both disambiguates weight
// modifier lines from priced items and gates item detection.
func classify(line string) (lineType, bool) {
	_, _, hasPrice := lastPrice(line)
	norm := normalizeName(line)

	switch {
	case norm == "":
		return lineNoise, hasPrice
	case containsAny(norm, totalKeywords):
		return lineTotal, hasPrice
	case containsAny(norm, depositKeywords):
		return lineDeposit, hasPrice
	case containsAny(norm, discountKeyword) || strings.HasPrefix(strings.TrimSpace(line), "-"):
		return lineDiscount, hasPrice
	case containsAny(norm, footerKeywords) || longNumberRe.MatchString(line):
		return lineFooter, hasPrice
	case strings.Contains(line, ":"):
		// Labels (OSSZEG:, AUTH:, D:...) carry a colon; sale lines do not.
		return lineFooter, hasPrice
	case ftPerKgRe.MatchString(line) || (strings.Contains(norm, "KG") && !hasPrice):
		return lineWeight, hasPrice
	case hasPrice && countLetters(line) >= 2:
		return lineItem, hasPrice
	case !hasPrice && countLetters(line) >= 4:
		// A descriptive line with no price: candidate name for a weighted item
		// whose price lands on the next line.
		return lineName, hasPrice
	default:
		return lineNoise, hasPrice
	}
}

// isFinalTotal distinguishes the grand total from a subtotal (RESZOSSZESEN).
func isFinalTotal(line string) bool {
	norm := normalizeName(line)
	if strings.Contains(norm, "RESZOSSZESEN") || strings.Contains(norm, "RESZOSSZ") {
		return false
	}
	return containsAny(norm, totalKeywords)
}
