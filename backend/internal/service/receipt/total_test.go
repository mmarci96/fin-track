package receipt

import "testing"

// Among several total-ish lines, the value closest to the item sum is chosen and
// a garbled payment line (0) is ignored.
func TestChooseTotalClosestToSum(t *testing.T) {
	lines := []string{
		"RESZOSSZESEN:\t5 360 Ft",
		"OSSZESEN\t5 360 Ft",
		"BANKKARTYA:\t0 Jol Et",
	}
	cands := collectTotalCandidates(lines)
	if got := chooseTotal(cands, 5345); got != 5360 {
		t.Fatalf("chooseTotal = %d, want 5360 (cands=%+v)", got, cands)
	}
}

// When the grand-total line is OCR-garbled, a clean payment line closest to the
// item sum wins.
func TestChooseTotalRecoversFromGarbledGrandTotal(t *testing.T) {
	lines := []string{
		"OSSZESEN\t9 360 Ft", // OCR'd "5 360" as "9 360"
		"BANKKARTYA:\t5 360 Ft",
	}
	cands := collectTotalCandidates(lines)
	if got := chooseTotal(cands, 5360); got != 5360 {
		t.Fatalf("chooseTotal = %d, want 5360 (cands=%+v)", got, cands)
	}
}

// With no candidate near the sum, a grand-total line is preferred for display.
func TestChooseTotalFallsBackToGrandTotal(t *testing.T) {
	lines := []string{
		"RESZOSSZESEN:\t1 000 Ft",
		"OSSZESEN\t2 000 Ft",
	}
	cands := collectTotalCandidates(lines)
	if got := chooseTotal(cands, 9999); got != 2000 {
		t.Fatalf("chooseTotal = %d, want 2000 (grand-total fallback)", got)
	}
}
