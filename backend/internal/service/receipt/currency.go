package receipt

import "regexp"

var (
	hufRe = regexp.MustCompile(`(?i)\b\d{1,3}(?:\s+\d{3})*(?:[.,]\d{2})?\s*(?:ft|huf)\b`)
	eurRe = regexp.MustCompile(`(?i)\b\d{1,3}(?:\s+\d{3})*(?:[.,]\d{2})?\s*(?:eur|€)\b`)
)

type Currency string

const (
	HUF Currency = "HUF"
	EUR Currency = "EUR"
)

func detectCurrency(lines []string) Currency {
	var hufCount, eurCount int

	for _, line := range lines {
		hufCount += len(hufRe.FindAllString(line, -1))
		eurCount += len(eurRe.FindAllString(line, -1))
	}

	// HUF wins ties
	if hufCount >= eurCount && hufCount > 0 {
		return HUF
	}

	if eurCount > 0 {
		return EUR
	}

	// For now return HUF even if nothing found
	return HUF
}
