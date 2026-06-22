package receipt

import (
	"context"
	"strings"
	"testing"
)

// OTP-restaurant ("OTP étterem") receipts use a two-line item layout:
//
//	0,332 KG X 5 150 Ft        <- quantity x unit price (modifier, not an item)
//	EBED 5150FT /  1 710 COO   <- item name + the real line total in a tab column
//
// These fixtures are golden OCR captures (data/IMG_097x.jpeg via the layout
// pipeline). They guard against the regressions where the heuristics emitted the
// per-unit price, the masked card PAN, or garbled separator lines as items.
func parseOTP(t *testing.T, fixture string) Result {
	t.Helper()
	return NewParser(canonicalMerchants, nil, nil).
		Parse(context.Background(), loadFixture(t, fixture))
}

func priceOf(items []Item, sub string) (int, bool) {
	for _, it := range items {
		if strings.Contains(strings.ToLower(it.Name), strings.ToLower(sub)) {
			return it.Price, true
		}
	}
	return 0, false
}

func TestParseOTP0973(t *testing.T) {
	r := parseOTP(t, "otp_0973.txt")

	if r.Total != 3385 {
		t.Errorf("total: got %d, want 3385", r.Total)
	}
	for _, want := range []struct {
		sub   string
		price int
	}{
		{"B2 MENU", 2330},
		{"EBED", 690},
		{"COCA-COLA", 315},
	} {
		if p, ok := priceOf(r.Items, want.sub); !ok {
			t.Errorf("missing item %q", want.sub)
		} else if p != want.price {
			t.Errorf("item %q: price got %d, want %d", want.sub, p, want.price)
		}
	}
	assertNoJunkItems(t, r.Items)
}

func TestParseOTP0975(t *testing.T) {
	r := parseOTP(t, "otp_0975.txt")

	if r.Total != 3341 {
		t.Errorf("total: got %d, want 3341", r.Total)
	}
	if p, ok := priceOf(r.Items, "FELAR"); !ok || p != 580 {
		t.Errorf("FELAR: got %d ok=%v, want 580", p, ok)
	}
	if p, ok := priceOf(r.Items, "EBED"); !ok || p != 1051 {
		t.Errorf("EBED: got %d ok=%v, want 1051", p, ok)
	}
	assertNoJunkItems(t, r.Items)
}

func TestParseOTP0972(t *testing.T) {
	// The printed total is OCR-garbled ("BESTESEN"), so it is not detected; what
	// matters is that the lone real item is captured and the masked card PAN
	// ("6101 32%* tt 4527") is not, which previously inflated the total to 7287.
	r := parseOTP(t, "otp_0972.txt")

	if p, ok := priceOf(r.Items, "A1 MENU"); !ok || p != 1380 {
		t.Errorf("A1 MENU: got %d ok=%v, want 1380", p, ok)
	}
	if len(r.Items) != 1 {
		t.Errorf("expected exactly 1 item, got %d: %+v", len(r.Items), r.Items)
	}
	assertNoJunkItems(t, r.Items)
}

// assertNoJunkItems checks the heuristics did not emit the per-unit price
// (5150 Ft/kg), a masked card PAN, or quantity-modifier lines as items.
func assertNoJunkItems(t *testing.T, items []Item) {
	t.Helper()
	for _, it := range items {
		if it.Price == 5150 {
			t.Errorf("emitted per-unit price as an item: %+v", it)
		}
		if strings.ContainsAny(it.Name, "*#%") {
			t.Errorf("emitted masked card PAN as an item: %+v", it)
		}
		if qtyUnitRe.MatchString(it.Name) {
			t.Errorf("emitted a quantity modifier as an item: %+v", it)
		}
	}
}
