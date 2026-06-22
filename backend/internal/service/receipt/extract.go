package receipt

import (
	"regexp"
	"strings"
)

// leadingCodeRe matches an Aldi-style 3-character product column code at the
// start of an item name (e.g. "COO Peroni", "000 Kockazott").
var leadingCodeRe = regexp.MustCompile(`^[A-Z0-9]{3}\s+`)

// minItemPrice filters out implausibly small prices (OCR noise like a stray "1"
// or an address house number). No real HUF grocery line costs under 10.
const minItemPrice = 10

// extracted holds the raw output of heuristic extraction before scoring.
type extracted struct {
	Items    []Item
	Total    int
	Currency string
}

// extractItems walks the cleaned lines and pulls out line items and the printed
// total. It handles two common layouts:
//
//   - name + trailing price on one line (Aldi, Rossmann name lines)
//   - a name line with no price followed by a weight line that carries the
//     price ("0,032 kg x 1 799 Ft/kg  957")
func extractItems(lines []string) extracted {
	var out extracted
	pendingName := ""

	for _, line := range lines[itemsStart(lines):] {
		t, hasPrice := classify(line)

		switch t {
		case lineTotal:
			println("Printing line for debug: ")
			if price, _, ok := lastPrice(line); ok {
				if isFinalTotal(line) || out.Total == 0 {
					out.Total = price
				}
			}
			pendingName = ""

		case lineWeight:
			// A weight line may carry the price for the preceding nameless item.
			// A weight line without a price (e.g. a tara line) is kept pending so
			// a following weight line can still pair with the name.
			if price, _, ok := lastPrice(line); ok && pendingName != "" && price >= minItemPrice {
				out.Items = append(out.Items, Item{Name: cleanItemName(pendingName), Price: price})
				pendingName = ""
			}

		case lineItem:
			price, rest, ok := lastPrice(line)
			name := cleanItemName(rest)
			if ok && name != "" && price >= minItemPrice {
				out.Items = append(out.Items, Item{Name: name, Price: price})
				pendingName = ""
			}

		case lineName:
			// Name with no price; the price may be on a following weight line.
			pendingName = cleanItemName(line)

		default:
			// header / footer / discount / deposit / noise: not an item, and they
			// break any pending name pairing.
			_ = hasPrice
			pendingName = ""
		}
	}

	return out
}

// itemsStart returns the index after the store/header block. Hungarian receipts
// print the tax id ("ADOSZAM") as the last header line before the items, so we
// skip everything up to and including it. Falls back to the top when absent.
func itemsStart(lines []string) int {
	for i, line := range lines {
		if strings.Contains(normalizeName(line), "ADOSZAM") {
			return i + 1
		}
	}
	return 0
}

// cleanItemName strips leading column codes/qty noise and trailing separators
// from an item name so it reads cleanly.
func cleanItemName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Trim(name, "-_.,| ")
	// Drop a leading OCR column code only when real text remains.
	if stripped := leadingCodeRe.ReplaceAllString(name, ""); countLetters(stripped) >= 4 {
		name = stripped
	}
	return strings.TrimSpace(name)
}
