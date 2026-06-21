package receipt

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var canonicalMerchants = []string{
	"ALDI MAGYARORSZAG ELELMISZER Bt.",
	"ROSSMANN MAGYARORSZAG KFT",
}

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}

func hasItemContaining(items []Item, sub string) bool {
	for _, it := range items {
		if strings.Contains(strings.ToLower(it.Name), strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

func hasWarning(ws []string, w string) bool {
	for _, x := range ws {
		if x == w {
			return true
		}
	}
	return false
}

func parse(t *testing.T, fixture string) Result {
	t.Helper()
	return NewParser(canonicalMerchants, nil).Parse(context.Background(), loadFixture(t, fixture))
}

func TestParseAldi0955(t *testing.T) {
	r := parse(t, "aldi_0955.txt")

	if !r.MerchantKnown || r.MerchantName != "ALDI MAGYARORSZAG ELELMISZER Bt." {
		t.Errorf("merchant: got known=%v name=%q", r.MerchantKnown, r.MerchantName)
	}
	if r.Decision != DecisionBestEffort {
		t.Errorf("decision: got %s want best_effort", r.Decision)
	}
	if r.Total != 0 {
		t.Errorf("total: got %d want 0 (garbled OSSZEG line)", r.Total)
	}
	if !hasWarning(r.Warnings, WarnTotalUnverified) {
		t.Errorf("expected total_unverified warning, got %v", r.Warnings)
	}
	for _, it := range r.Items {
		if it.Price < minItemPrice {
			t.Errorf("item %q has sub-threshold price %d", it.Name, it.Price)
		}
		if strings.Contains(strings.ToLower(it.Name), "visszavalt") {
			t.Errorf("deposit line leaked as item: %q", it.Name)
		}
		if strings.Contains(it.Name, "MESZAROSOK") {
			t.Errorf("address line leaked as item: %q", it.Name)
		}
	}
	for _, want := range []string{"Maretti", "Peroni", "Burgonya"} {
		if !hasItemContaining(r.Items, want) {
			t.Errorf("expected an item containing %q; items=%v", want, r.Items)
		}
	}
}

func TestParseAldi0963(t *testing.T) {
	r := parse(t, "aldi_0963.txt")

	if !r.MerchantKnown {
		t.Errorf("garbled ALUL header should still match canonical ALDI; name=%q", r.MerchantName)
	}
	if r.Total != 9360 {
		t.Errorf("total: got %d want 9360", r.Total)
	}
	if r.Decision != DecisionBestEffort || !hasWarning(r.Warnings, WarnTotalMismatch) {
		t.Errorf("expected best_effort + total_mismatch; got %s %v", r.Decision, r.Warnings)
	}
	// Weighted produce: name on one line, price on the following weight line.
	if !hasItemContaining(r.Items, "Paradicsom") {
		t.Errorf("expected weighted item 'Paradicsom' to be recovered; items=%v", r.Items)
	}
	for _, want := range []string{"Napraforgó", "zacskó"} {
		if !hasItemContaining(r.Items, want) {
			t.Errorf("expected an item containing %q; items=%v", want, r.Items)
		}
	}
}

func TestParseRossmann0967(t *testing.T) {
	r := parse(t, "rossmann_0967.txt")

	if !r.MerchantKnown || r.MerchantName != "ROSSMANN MAGYARORSZAG KFT" {
		t.Errorf("merchant: got known=%v name=%q", r.MerchantKnown, r.MerchantName)
	}
	if r.Total != 43721 {
		t.Errorf("total: got %d want 43721", r.Total)
	}
	if len(r.Items) < 20 {
		t.Errorf("expected >=20 items, got %d", len(r.Items))
	}
	for _, it := range r.Items {
		if strings.HasPrefix(it.Name, "EAN") || strings.HasPrefix(it.Name, "FAN") {
			t.Errorf("EAN/barcode line leaked as item: %q", it.Name)
		}
		if it.Price < 0 {
			t.Errorf("discount leaked as negative item: %q", it.Name)
		}
	}
	if !hasItemContaining(r.Items, "MELATONIN") {
		t.Errorf("expected MELATONIN item; items=%v", r.Items)
	}
}

func TestParseRejectsUnreadable(t *testing.T) {
	r := NewParser(canonicalMerchants, nil).Parse(context.Background(),
		"some blurry\ntext with no\nprices at all\n")
	if r.Decision != DecisionReject {
		t.Errorf("expected reject for unreadable input, got %s with %d items", r.Decision, len(r.Items))
	}
}

func TestParseNeverPanicsOnEmptyKnownList(t *testing.T) {
	// Regression for the old findBestMatch index-out-of-range crash.
	r := NewParser(nil, nil).Parse(context.Background(), loadFixture(t, "aldi_0955.txt"))
	if r.MerchantKnown {
		t.Errorf("no known merchants => should be unknown")
	}
	if len(r.Items) == 0 {
		t.Errorf("should still extract items for an unknown merchant")
	}
	if !hasWarning(r.Warnings, WarnMerchantUnknown) {
		t.Errorf("expected merchant_unknown warning, got %v", r.Warnings)
	}
}

// stubLLM returns a fixed extraction, simulating the Ollama fallback.
type stubLLM struct {
	items []Item
	total int
}

func (s stubLLM) ExtractItems(context.Context, string) ([]Item, int, error) {
	return s.items, s.total, nil
}

func TestLLMFallbackAdoptedWhenHeuristicEmpty(t *testing.T) {
	fb := stubLLM{items: []Item{{Name: "Bread", Price: 500}, {Name: "Milk", Price: 300}}, total: 800}
	r := NewParser(canonicalMerchants, fb).Parse(context.Background(),
		"unreadable\ngarbage\nlines\n")

	if len(r.Items) != 2 {
		t.Fatalf("expected fallback items adopted, got %d", len(r.Items))
	}
	if r.Decision == DecisionReject {
		t.Errorf("with usable fallback items, should not reject")
	}
	if !hasWarning(r.Warnings, WarnLLMFallbackUsed) {
		t.Errorf("expected llm_fallback_used warning, got %v", r.Warnings)
	}
}
