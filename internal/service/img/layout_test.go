package img

import "testing"

func TestReconstructLayoutInsertsColumnTab(t *testing.T) {
	// Name words sit close together; the price sits far to the right.
	words := []wordBox{
		{text: "Maretti", block: 1, par: 1, line: 1, minX: 10, maxX: 80, top: 10},
		{text: "70", block: 1, par: 1, line: 1, minX: 85, maxX: 100, top: 10},
		{text: "399", block: 1, par: 1, line: 1, minX: 300, maxX: 340, top: 10},
	}
	got := reconstructLayout(words)
	want := "Maretti 70\t399"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestReconstructLayoutOrdersLinesTopDown(t *testing.T) {
	// Provided out of vertical order; should be emitted top-to-bottom.
	words := []wordBox{
		{text: "second", block: 1, par: 1, line: 2, minX: 10, maxX: 70, top: 100},
		{text: "first", block: 1, par: 1, line: 1, minX: 10, maxX: 70, top: 10},
	}
	got := reconstructLayout(words)
	want := "first\nsecond"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestReconstructLayoutEmpty(t *testing.T) {
	if got := reconstructLayout(nil); got != "" {
		t.Errorf("got %q want empty", got)
	}
}
