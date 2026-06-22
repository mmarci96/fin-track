package receipt

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	// Rossmann prints a column code (C00 / COO / coo, often with a trailing pipe)
	// at the end of item lines. OCR mangles it, so match loosely.
	trailingColumnCodeRe = regexp.MustCompile(`(?i)\s+c[o0]{2}\b\s*\|?\s*$`)
	// Leftover separators OCR leaves at line ends.
	trailingJunkRe = regexp.MustCompile(`[\s|©"'\x60]+$`)
	leadingJunkRe  = regexp.MustCompile(`^[\s|©"'.\x60~]+`)

	// A price at the end of a line: an optional thousands-grouped number
	// ("1 099", "10 984") or a plain run of digits, optionally followed by "Ft".
	endPriceRe = regexp.MustCompile(`(?:^|\s)(\d{1,3}(?:\s\d{3})+|\d+)\s*(?:Ft)?\s*$`)
	// The first price-looking token in a column: a thousands-grouped number or a
	// run of 2+ digits. Used for tab columns where the line total is followed by
	// an OCR'd tax-category code ("690 COO", "1 380 £00", "315 :C00").
	firstPriceRe = regexp.MustCompile(`\d{1,3}(?:\s\d{3})+|\d{2,}`)
)

// normalizeText splits OCR output into cleaned, non-empty lines.
func normalizeText(text string) []string {
	raw := strings.Split(text, "\n")
	lines := make([]string, 0, len(raw))
	for _, l := range raw {
		if c := cleanLine(l); c != "" {
			lines = append(lines, c)
		}
	}
	return lines
}

// cleanLine trims a single line and strips trailing OCR column codes and junk.
func cleanLine(line string) string {
	line = strings.TrimSpace(line)
	line = trailingColumnCodeRe.ReplaceAllString(line, "")
	line = trailingJunkRe.ReplaceAllString(line, "")
	line = leadingJunkRe.ReplaceAllString(line, "")
	return strings.TrimSpace(line)
}

// lastPrice returns the trailing price on a line and the remaining text before
// it. ok is false when no trailing price token is present.
//
// When the line carries column separators (tabs inserted by the layout-aware
// OCR step), the price is taken from the rightmost column that contains one.
// This prevents stray digits in the name column from being glued onto the
// price ("Maretti 70 9 \t 399" -> 399, not 9399).
func lastPrice(line string) (price int, rest string, ok bool) {
	if strings.Contains(line, "\t") {
		cols := strings.Split(line, "\t")
		for i := len(cols) - 1; i >= 0; i-- {
			if p, found := firstPriceIn(cols[i]); found {
				rest = strings.TrimSpace(strings.Join(cols[:i], " "))
				return p, rest, true
			}
		}
		return 0, strings.TrimSpace(strings.ReplaceAll(line, "\t", " ")), false
	}
	return priceAtEnd(line)
}

// firstPriceIn returns the first price-looking token in a single column. The
// price column carries the line total first, sometimes followed by a garbled
// tax-category code ("690 COO" -> 690, "1 380 £00" -> 1380).
func firstPriceIn(col string) (price int, ok bool) {
	m := firstPriceRe.FindString(col)
	if m == "" {
		return 0, false
	}
	val, err := parseAmount(m)
	if err != nil {
		return 0, false
	}
	return val, true
}

// priceAtEnd finds a trailing price token within a single column of text.
func priceAtEnd(line string) (price int, rest string, ok bool) {
	m := endPriceRe.FindStringSubmatchIndex(line)
	if m == nil {
		return 0, line, false
	}
	token := line[m[2]:m[3]]
	val, err := parseAmount(token)
	if err != nil {
		return 0, line, false
	}
	rest = strings.TrimSpace(line[:m[2]])
	return val, rest, true
}

// parseAmount parses an integer amount, ignoring thousands spaces.
func parseAmount(s string) (int, error) {
	return strconv.Atoi(strings.ReplaceAll(s, " ", ""))
}

// NormalizeName exposes normalizeName so other packages (e.g. the repository)
// compute the exact same merchant de-duplication key.
func NormalizeName(s string) string { return normalizeName(s) }

// normalizeName folds a merchant/string to an accent-free, uppercase,
// punctuation-free key for matching and de-duplication.
func normalizeName(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch r {
		case 'Á':
			r = 'A'
		case 'É':
			r = 'E'
		case 'Í':
			r = 'I'
		case 'Ó', 'Ö', 'Ő':
			r = 'O'
		case 'Ú', 'Ü', 'Ű':
			r = 'U'
		}
		if r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			continue
		}
		if r == ' ' {
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
