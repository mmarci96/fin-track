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
	lineUnitQty // "<qty> <unit> X <unit price>": modifier preceding a named item
	lineDiscount
	lineDeposit
	lineTotal
	lineFooter
)

var (
	longNumberRe = regexp.MustCompile(`\d{8,}`)
	ftPerKgRe    = regexp.MustCompile(`(?i)FT\s*/\s*KG`)
	// qtyUnitRe matches a quantity line that states "<qty> <unit> X <unit price>"
	// (e.g. "0,332 KG X 5 150 Ft", "1 DB X 580 Ft", "1 ADAG X 2 330 Ft"). The
	// trailing number is the per-unit price, never the line total, so these must
	// not be emitted as their own items.
	qtyUnitRe = regexp.MustCompile(`(?i)^\s*\d[\d.,\s]*(?:KG|DKG|DB|ADAG)\s*X\b`)
	// maskedCardRe matches a masked card PAN ("4229 87** **** 5080", OCR'd
	// variously as "32%* tt", "92*% #4%%") printed in the payment footer; it must
	// never be read as an item.
	maskedCardRe = regexp.MustCompile(`[*#%]{2,}`)
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
	case containsAny(norm, footerKeywords) || longNumberRe.MatchString(line) || maskedCardRe.MatchString(line):
		return lineFooter, hasPrice
	case strings.Contains(strings.SplitN(line, "\t", 2)[0], ":"):
		// Labels (OSSZEG:, AUTH:, D:...) carry a colon in their leading column;
		// sale lines do not. Only inspect the text before the first tab so a
		// trailing tax-code in the price column (e.g. "315 :C00") is not mistaken
		// for a label.
		return lineFooter, hasPrice
	case qtyUnitRe.MatchString(line):
		// "<qty> <unit> X ...". When the per-kg rate is on this same line (Aldi:
		// "0,032 kg x 1 799 Ft/kg 957") the trailing number is the line total, so
		// treat it as a weight line that pairs with the preceding name. Otherwise
		// (OTP: "0,332 KG X 5 150 Ft") the trailing number is only the unit price
		// and the named item with its real total follows on the next line.
		if ftPerKgRe.MatchString(line) {
			return lineWeight, hasPrice
		}
		return lineUnitQty, hasPrice
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
