package img

import (
	"sort"
	"strings"

	"github.com/otiai10/gosseract/v2"
)

// gapCharMultiplier controls how wide a horizontal gap (relative to the average
// character width) must be before it is treated as a column boundary rather than
// a normal inter-word space. Receipts put the price in a far-right column, so a
// large gap there becomes a tab the parser can key on.
const gapCharMultiplier = 2.5

// wordBox is a single OCR word with its position. It decouples the layout logic
// from the gosseract type so it can be unit tested without running OCR.
type wordBox struct {
	text             string
	block, par, line int
	minX, maxX, top  int
}

func boxesToWords(boxes []gosseract.BoundingBox) []wordBox {
	words := make([]wordBox, 0, len(boxes))
	for _, b := range boxes {
		t := strings.TrimSpace(b.Word)
		if t == "" {
			continue
		}
		words = append(words, wordBox{
			text:  t,
			block: b.BlockNum,
			par:   b.ParNum,
			line:  b.LineNum,
			minX:  b.Box.Min.X,
			maxX:  b.Box.Max.X,
			top:   b.Box.Min.Y,
		})
	}
	return words
}

func avgCharWidth(words []wordBox) float64 {
	totalW, totalC := 0, 0
	for _, w := range words {
		totalW += w.maxX - w.minX
		totalC += len([]rune(w.text))
	}
	if totalC == 0 {
		return 0
	}
	return float64(totalW) / float64(totalC)
}

type lineKey struct{ block, par, line int }

// reconstructLayout rebuilds text from word boxes, joining words on a line with
// a single space, but inserting a tab where a wide horizontal gap signals a
// column boundary. Lines are ordered top-to-bottom.
func reconstructLayout(words []wordBox) string {
	if len(words) == 0 {
		return ""
	}
	threshold := avgCharWidth(words) * gapCharMultiplier

	groups := map[lineKey][]wordBox{}
	order := []lineKey{}
	minTop := map[lineKey]int{}
	for _, w := range words {
		k := lineKey{w.block, w.par, w.line}
		if _, ok := groups[k]; !ok {
			order = append(order, k)
			minTop[k] = w.top
		}
		groups[k] = append(groups[k], w)
		if w.top < minTop[k] {
			minTop[k] = w.top
		}
	}

	sort.SliceStable(order, func(i, j int) bool {
		return minTop[order[i]] < minTop[order[j]]
	})

	var sb strings.Builder
	for li, k := range order {
		ws := groups[k]
		sort.SliceStable(ws, func(i, j int) bool { return ws[i].minX < ws[j].minX })

		if li > 0 {
			sb.WriteByte('\n')
		}
		for wi, w := range ws {
			if wi > 0 {
				gap := w.minX - ws[wi-1].maxX
				if threshold > 0 && float64(gap) > threshold {
					sb.WriteByte('\t')
				} else {
					sb.WriteByte(' ')
				}
			}
			sb.WriteString(w.text)
		}
	}
	return sb.String()
}
