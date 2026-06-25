package receipt

import "testing"

var knownTwo = []string{"ALDI MAGYARORSZAG ELELMISZER Bt.", "ROSSMANN MAGYARORSZAG KFT"}

// A header whose brand matches no known merchant (and whose full string is not
// similar enough) stays unknown — but a learned alias for the variant resolves it.
func TestDetectMerchantResolvesViaAlias(t *testing.T) {
	header := "OTP BANK NYRT ALTAL KEPVISELT CSOPORTOS"
	lines := []string{header, "1051 BUDAPEST"}

	if mm := detectMerchant(lines, knownTwo, nil); mm.Known {
		t.Fatalf("expected unknown without alias, got canonical=%q score=%.2f", mm.Canonical, mm.Score)
	}

	aliases := []MerchantAlias{{Normalized: normalizeName(header), Canonical: knownTwo[0]}}
	mm := detectMerchant(lines, knownTwo, aliases)
	if !mm.Known || mm.Canonical != knownTwo[0] {
		t.Fatalf("expected known via alias, got known=%v canonical=%q", mm.Known, mm.Canonical)
	}
}

// The shared "MAGYARORSZAG KFT" tail must not make AUCHAN look like ROSSMANN.
func TestDetectMerchantRejectsSuffixOnlyMatch(t *testing.T) {
	lines := []string{"AUCHAN MAGYARORSZAG Kft.", "2040 BUDAORS"}
	if mm := detectMerchant(lines, knownTwo, nil); mm.Known {
		t.Fatalf("AUCHAN should not match a known merchant, got canonical=%q score=%.2f", mm.Canonical, mm.Score)
	}
}

// A garbled-but-intact ALDI legal name still resolves via whole-string similarity.
func TestDetectMerchantMatchesGarbledBrand(t *testing.T) {
	lines := []string{"ALUL MAGYAROROZAG ELELMISZER Bt,", "2051 BIATORBAGY"}
	mm := detectMerchant(lines, knownTwo, nil)
	if !mm.Known || mm.Canonical != knownTwo[0] {
		t.Fatalf("expected ALDI, got known=%v canonical=%q score=%.2f", mm.Known, mm.Canonical, mm.Score)
	}
}

// A header with only a postal code + city (no brand) is unknown — the alias
// flywheel is meant to resolve these once the header is corrected and approved.
func TestDetectMerchantCityOnlyIsUnknown(t *testing.T) {
	lines := []string{"2051. BIATORBAGY", "101, sz. uzlet"}
	if mm := detectMerchant(lines, knownTwo, nil); mm.Known {
		t.Fatalf("city-only header should be unknown, got canonical=%q score=%.2f", mm.Canonical, mm.Score)
	}
}
