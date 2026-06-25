# Receipt parsing reference (OCR text → stored model)

A stage-by-stage map of how raw OCR text becomes a stored `Receipt`, with every
regex / keyword set / decision rule, ✓/✗ examples, and **how to verify each rule
in a unit test**. Every function below is pure and package-internal, so tests in
`package receipt` can call them directly — no DB, no HTTP.

> Scope: this is a *review* document. The "Known gaps" section lists candidate
> rules that are **not implemented** (for discussion before any change).

## Pipeline at a glance

```
OCR text (string, tab-separated columns)
  │
  ▼ normalizeText            normalize.go   → []string (cleaned, non-empty lines)
  ▼ detectMerchant           merchant.go    → merchant_name, merchant_known
  ▼ detectCurrency           currency.go    → HUF / EUR
  ▼ extractItems             extract.go     → items[], depositSum, discountSum
  ▼ collectTotalCandidates   total.go       → candidate totals
  ▼ chooseTotal              total.go       → printed total
  ▼ reconcile (inline)       parse.go       → computed_total, reconciled
  ▼ decide                   parse.go       → confidence, decision, warnings
  │  (optional LLM fallback when heuristics are weak)
  ▼ Result                   types.go
  ▼ toModelReceipt           handler/image.go → model.Receipt (+ merchant resolve)
  ▼ CreateReceipt            repository       → DB rows
```

Driver: `Parser.Parse` (`parse.go`). The HTTP entry is `ImageHandler.parseText`
(`handler/image.go`), which loads verified merchants + currencies + learned
aliases and calls `Parse`.

---

## Stage 1 — Normalize (`normalize.go`)

`normalizeText(text)` splits on `\n`, runs `cleanLine` on each, drops empties.

### `cleanLine` — trims a line and strips trailing OCR cruft
| regex | purpose | ✓ matches (stripped) | ✗ leaves alone |
|---|---|---|---|
| `trailingColumnCodeRe` = `(?i)\s+c[o0]{2}\b\s*\|?\s*$` | Rossmann column code `C00`/`COO` (+ optional `\|`) at line end | `Maretti 399 C00` → `Maretti 399`; `Sajt 690 coo\|` | `C00 Peroni` (leading, not trailing) |
| `trailingJunkRe` = `[\s\|©"'` + "`" + `]+$` | trailing separators / quotes | `Tej 299 "` → `Tej 299` | `Tej "Bio" 299` |
| `leadingJunkRe` = `^[\s\|©"'.` + "`" + `~]+` | leading separators | `. Kenyer 451` → `Kenyer 451` | `1211 Budapest` |

**Verify:** `cleanLine("Maretti 399 C00")` == `"Maretti 399"`.

### `normalizeName` — accent-free, UPPERCASE, alnum+space key
Used for keyword matching and merchant de-dup. Folds `Á→A É→E Í→I Ó/Ö/Ő→O Ú/Ü/Ű→U`,
keeps only `A–Z 0–9` and space, collapses whitespace via `strings.Fields`.

> ⚠️ **Non-alnum is dropped, not spaced.** `/`, `.`, and **tab** become nothing,
> so `2026/059196` → `2026059196` and `…196\t11.400` glues into one long digit
> run. This matters in the worked example below.

| input | output |
|---|---|
| `ÖSSZESEN:` | `OSSZESEN` |
| `ALDI MAGYARORSZÁG ELELMISZER Bt.` | `ALDI MAGYARORSZAG ELELMISZER BT` |
| `PES NY 2026/059196\t11.400 Ft` | `PES NY 202605919611400 FT` |

**Verify:** `normalizeName("ÖSSZESEN:")` == `"OSSZESEN"`.

### Price reading — `lastPrice` (and helpers)
Returns `(price, restText, ok)`.
- **Tab lines** (`\t` present): split on tab, scan columns right→left, take the
  first column whose `firstPriceIn` matches. Prevents stray name-column digits
  from gluing onto the price.
- **Non-tab lines**: `priceAtEnd` uses the trailing-price regex.

| regex | used by | purpose |
|---|---|---|
| `endPriceRe` = `(?:^\|\s)(\d{1,3}(?:\s\d{3})+\|\d+)\s*(?:Ft)?\s*$` | `priceAtEnd` | trailing price, **space**-grouped thousands or plain run, optional `Ft` |
| `firstPriceRe` = `\d{1,3}(?:\s\d{3})+\|\d{2,}` | `firstPriceIn` | first price token in one tab column (≥2 digits) |
| `parseAmount` | both | `strconv.Atoi` after removing **spaces** only |

> ⚠️ **Only space-grouping is understood.** `1 099` → 1099 ✓, but `11.400`
> (period thousands) is **not** handled: `parseAmount`/regexes treat `.` as a
> break, so `11.400` reads as `11`. See Known gaps.

| line | `lastPrice` → price | note |
|---|---|---|
| `Maretti\t399` | 399 | tab, rightmost col |
| `Tej 1 099 Ft` | 1099 | space thousands ✓ |
| `Maretti 70 9 \t 399` | 399 | name-col digits ignored ✓ |
| `…\t11.400 Ft` | **11** | period thousands ✗ (should be 11400) |

**Verify:** `p, _, ok := lastPrice("Tej 1 099 Ft")` → `ok && p == 1099`.

---

## Stage 2 — Merchant (`merchant.go`)

`detectMerchant(lines, known, aliases)` scans the first `headerScanLines` (6)
lines and returns `{Canonical, Candidate, Score, Known}`.

A header matches a known (verified) merchant when **any** of:
- **(a) brand-token** similarity ≥ `brandThreshold` (0.7): each distinctive header
  token vs each known merchant's brand token.
- **(b) whole-string** similarity ≥ `strongThreshold` (0.8): full normalized
  header vs full normalized known name.
- **(c) alias**: full normalized header vs a learned `MerchantAlias.Normalized`
  ≥ `merchantThreshold` (0.6).

| helper | purpose |
|---|---|
| `brandTokens(name)` | normalized tokens minus `merchantStopwords` (`KFT,ZRT,BT,NYRT,RT,KKT,KERESKEDELMI,ELELMISZER,MAGYARORSZAG,HUNGARY,ES`) and pure-digit tokens |
| `brandSimilarity(a,b)` | `similarity`, but tokens shorter than `minBrandLen` (3) must match exactly |
| `similarity(a,b)` | `1 − levenshtein/maxLen` |

Why two thresholds: the shared corporate tail `MAGYARORSZAG KFT` inflates
whole-string similarity, so `AUCHAN…KFT` vs `ROSSMANN…KFT` ≈ 0.72 must be rejected
(< 0.8), while a garbled-but-intact `ALUL MAGYAROROZAG…` vs `ALDI MAGYARORSZAG…`
≈ 0.87 must pass.

| header (known = ALDI, ROSSMANN) | result | via |
|---|---|---|
| `ALDI MAGYARORSZÁG ELELMISZER Bt.` | known → ALDI | (a) brand `ALDI` |
| `ALUL MAGYAROROZAG ELELMISZER Bt,` | known → ALDI | (b) whole-string 0.87 |
| `AUCHAN MAGYARORSZAG Kft.` | **unknown** | (a) 0.13, (b) 0.72 |
| `2051. BIATORBAGY` | **unknown** | no brand, low full sim |
| `OTP BANK NYRT …` + alias | known → ALDI | (c) alias |

**Verify:** `detectMerchant([]string{"AUCHAN MAGYARORSZAG Kft."}, []string{"ALDI…","ROSSMANN…"}, nil).Known == false`.
(See `merchant_alias_test.go` for the live cases.)

---

## Stage 3 — Currency (`currency.go`)

`detectCurrency` counts HUF vs EUR markers; HUF wins ties and the empty case.

| regex | matches |
|---|---|
| `hufRe` = `(?i)\b\d{1,3}(?:\s+\d{3})*(?:[.,]\d{2})?\s*(?:ft\|huf)\b` | `1 099 Ft`, `250 HUF` |
| `eurRe` = `(?i)\b\d{1,3}(?:\s+\d{3})*(?:[.,]\d{2})?\s*(?:eur\|€)\b` | `3,50 EUR`, `3.50€` |

**Verify:** `detectCurrency([]string{"OSSZESEN 1 099 Ft"})` == `HUF`.

---

## Stage 4 — Item extraction (`classify.go` + `extract.go`)

### `itemsStart(lines)`
Returns the index after the `ADOSZAM` (tax-id) header line, else 0.

### `classify(line)` → `(lineType, hasPrice)` — **order matters**
Evaluated top-down; first match wins:

| # | branch | rule |
|---|---|---|
| 1 | `lineNoise` | `norm == ""` |
| 2 | `lineTotal` | `norm` contains any `totalKeywords` (`OSSZESEN, OSSZEG, FIZETENDO, VEGOSSZEG`) |
| 3 | `lineDeposit` | `norm` contains `depositKeywords` (`VISSZAVALT, BETETDIJ`) |
| 4 | `lineDiscount` | `norm` contains `discountKeyword` (`ENGEDMENY, AKCIO, KEDVEZMENY`) **or** `line` starts with `-` |
| 5 | `lineFooter` | `norm` contains `footerKeywords` (`BANKKARTYA, KESZPENZ, VISA, MASTERCARD, CONTACTLESS, AUTH, ADOSZAM, TERMINAL, AID, APP, EAN, FAN`) **or** `longNumberRe \d{8,}` on **`line`** **or** `maskedCardRe [*#%]{2,}` |
| 6 | `lineFooter` | text before first tab contains `:` (labels like `OSSZEG:`) |
| 7 | `lineUnitQty` / `lineWeight` | `qtyUnitRe` `(?i)^\s*\d[\d.,\s]*(?:KG\|DKG\|DB\|ADAG)\s*X\b`; with `ftPerKgRe` `(?i)FT\s*/\s*KG` on the line → weight, else unit-qty |
| 8 | `lineWeight` | `ftPerKgRe` or (`KG` in norm and `!hasPrice`) |
| 9 | `lineItem` | `hasPrice && countLetters(line) >= 2` |
| 10 | `lineName` | `!hasPrice && countLetters(line) >= 4` |
| 11 | `lineNoise` | default |

> ⚠️ Branch 5 tests `longNumberRe` against the **raw `line`** (separators intact),
> not `norm`. `2026/059196` has no 8-digit run in `line`, so it does not trip the
> footer rule — even though `norm` would (`202605919611400`). See worked example.

### `isFinalTotal(line)`
`true` for `totalKeywords` **except** `RESZOSSZESEN`/`RESZOSSZ` (subtotal). The
grand total ends the item section (extractItems returns).

### `extractItems(lines)` walk (from `itemsStart`)
| lineType | action |
|---|---|
| `lineTotal` | reset pending; if `isFinalTotal` → **return** (stop) |
| `lineDeposit` | `DepositSum += lastPrice` (if > 0) — a real charge, counts toward reconcile |
| `lineDiscount` | `DiscountSum += lastPrice` (if > 0) — subtracted |
| `lineWeight` | pair price with the preceding `pendingName` (if ≥ `minItemPrice`) |
| `lineItem` | emit `{cleanItemName(rest), price}` if name≠"" and price ≥ `minItemPrice` (10) |
| `lineUnitQty` | reset pending (trailing number is a unit price, not a total) |
| `lineName` | set `pendingName` |
| default | reset pending |

### `cleanItemName(name)`
| regex | purpose |
|---|---|
| `unitPriceSuffixRe` = `(?i)\s+\S*\d+\s*FT\s*/\s*\w*\.?$` | strip trailing per-unit suffix (`EBED 5150FT /` → `EBED`) |
| `leadingCodeRe` = `^[A-Z0-9]{3}\s+` | strip a leading column code **only if ≥4 letters remain** |
| `strings.Trim(name, "-_.,\| ")` | trim separators |

**Verify per branch** (table-driven recommended):
- `classify("OSSZESEN:\t5 360 Ft")` → `lineTotal`; `isFinalTotal(...)` → `true`.
- `classify("RESZOSSZESEN:\t5 360")` → `lineTotal`; `isFinalTotal` → `false`.
- `classify("0,332 KG X 5 150 Ft")` → `lineUnitQty`.
- `classify("0,032 kg x 1 799 Ft/kg 957")` → `lineWeight`.
- `classify("VISA **** 5080")` → `lineFooter`.
- `extractItems` on a small fixture → expected `Items` / `DepositSum` / `DiscountSum`.

---

## Stage 5 — Total selection (`total.go`)

`collectTotalCandidates(lines)` scans **all** lines, grouping any line with a
parseable `lastPrice > 0`:
| group | rule |
|---|---|
| `groupSubtotal` | `RESZOSSZESEN` / `RESZOSSZ` |
| `groupGrand` | other `totalKeywords` |
| `groupPayment` | `paymentKeywords` (`BANKKARTYA, KESZPENZ, KARTYA`) |

`chooseTotal(cands, target)` → candidate **closest to `target`**; if within
`reconcileTolerance(target)` use it, else prefer a `groupGrand` candidate, else
the closest; `0` if no candidates.

| candidates | target (item sum) | chosen |
|---|---|---|
| `{RESZ 5360, OSSZ 5360, BANK 0}` | 5345 | 5360 (0 ignored) |
| `{OSSZ 9360(garbled), BANK 5360}` | 5360 | 5360 |
| `{RESZ 1000, OSSZ 2000}` | 9999 | 2000 (grand fallback) |

**Verify:** see `total_test.go`.

---

## Stage 6 — Reconcile + decide (`parse.go`, `reconcile.go`)

In `Parse`:
- `computed = Σ item.Price + DepositSum − DiscountSum` → `res.ComputedTotal`
- `res.Total = chooseTotal(collectTotalCandidates(lines), computed)`
- `res.Reconciled = Total > 0 && abs(computed − Total) ≤ reconcileTolerance(Total)`
- `reconcileTolerance(t) = max(t/100, 50)` (1% band, 50 Ft floor)

`reconcile()` (item-only `ComputedTotal`) is still used by the **LLM fallback**
path (`adoptFallback`), which has no deposit/discount info.

`shouldFallback` triggers the LLM when: 0 items, or a printed total that does not
reconcile, or `< minItems` (2). `adoptFallback` adopts LLM output only if it
reconciles when the heuristic didn't, or yields more items.

`decide`:
- 0 items → `reject`, confidence 0.
- else warnings: `merchant_unknown`, `total_unverified` (Total 0) / `total_mismatch`
  (Total>0 not reconciled), `low_item_count` (<2).
- `accepted` iff reconciled **and** merchant known; otherwise `best_effort`.
- `confidence` = 0.25 known + 0.15 total>0 + 0.45 reconciled + 0.15 items≥2.

**Verify:** `reconcileTolerance(5360)` == 53; build a `Result` and call the public
`Parse` on a fixture, assert `Decision`/`Reconciled`/`Warnings`.

---

## Stage 7 — Map to model (`handler/image.go`)

`toModelReceipt(result, merchantID, userID)`:
- `total_amount = result.Total`, or `result.ComputedTotal` when `Total == 0`.
- `products[] = {name, price}` from items.
- `currency` from the detected code.

`persistReceipt`: merchant resolved **only when `MerchantKnown`** (canonical
verified name); unknown headers collapse to one `UNKNOWN` sentinel row (no junk
rows). Verified-merchant list comes from `FindVerifiedMerchants()`.

---

## Worked example — the line that slipped through

Raw (tabs shown as `→`): `PES NY 2026/059196→11.400 Ft C00`
Real total is on later lines: `OSSZESEN:→11 400 Ft`, `KESZPENZ→11 400 Ft`.

| step | result |
|---|---|
| `cleanLine` | strips ` C00` → `PES NY 2026/059196→11.400 Ft` |
| `lastPrice` | tab split `["PES NY 2026/059196","11.400 Ft"]`; `firstPriceIn("11.400 Ft")` = **11** (period breaks thousands) → `hasPrice=true, price=11, rest="PES NY 2026/059196"` |
| `classify` branch 5 | `longNumberRe \d{8,}` tested on **`line`** → longest run is `059196` (6) → **no match** (would match `norm` `202605919611400`) |
| `classify` branch 6 | text before tab `PES NY 2026/059196` has no `:` → not footer |
| `classify` branch 9 | `hasPrice && letters≥2` → **`lineItem`** |
| `cleanItemName` | `leadingCodeRe` strips `PES ` only if ≥4 letters remain; `NY 2026/059196` has 2 letters → **kept** → name `PES NY 2026/059196` |
| `extractItems` | price 11 ≥ 10 → **emits `{"PES NY 2026/059196", 11}`** |

So a *nyugtaszám* (receipt-number) line becomes a bogus 11 Ft item. Two distinct
root causes meet here: a serial line that looks like a priced item, and the
period-thousands misread.

---

## Known gaps / candidate rules (NOT implemented — for discussion)

1. **Period-thousands amounts** (`11.400` → should be 11400). The price regexes
   and `parseAmount` only understand **space** grouping. Candidate: accept
   `\d{1,3}(?:[.\s]\d{3})+` and strip `.`/space in `parseAmount` — but must not
   clobber decimal cents (`3,50`/`3.50` EUR). Needs a HUF-vs-decimal rule.
2. **`longNumberRe` tests `line`, not `norm`.** Serial/doc numbers with internal
   separators (`2026/059196`) dodge the footer rule. Candidate: also test `norm`,
   or add a receipt-number pattern (`(?i)\bNY\s*\d+\s*/\s*\d+\b`) as footer/noise.
3. **Serial-line shape `<letters> <digits>/<digits> <amount> C00`.** Could be a
   dedicated footer/noise classifier branch; risk of catching real items needs a
   fixture sweep.
4. **Multi-line deposits** (Auchan `VISSZAVALTASI DIJ` value on the next line) and
   **inline-negative discounts** (`Cheetos 2db -100` read as a +100 item) — both
   inflate the item sum. Deeper `extractItems` pairing work.

---

## Verification cookbook

Everything is a pure function in `package receipt`; put tests in
`internal/service/receipt/*_test.go` and run:

```
go test ./internal/service/receipt/...
```

Directly testable units (no DB/HTTP):
`normalizeName`, `cleanLine`, `lastPrice`, `firstPriceIn`, `priceAtEnd`,
`parseAmount`, `classify`, `isFinalTotal`, `itemsStart`, `cleanItemName`,
`extractItems`, `collectTotalCandidates`, `chooseTotal`, `detectMerchant`,
`brandTokens`, `brandSimilarity`, `similarity`, `detectCurrency`,
`reconcileTolerance`. End-to-end: `NewParser(known, currencies, nil).Parse(ctx, text)`.

Suggested table-driven skeleton:

```go
func TestClassify(t *testing.T) {
	cases := []struct{ in string; want lineType }{
		{"OSSZESEN:\t5 360 Ft", lineTotal},
		{"0,332 KG X 5 150 Ft", lineUnitQty},
		{"VISA **** 5080", lineFooter},
		{"PES NY 2026/059196\t11.400 Ft", lineItem}, // documents current (wrong) behavior
	}
	for _, c := range cases {
		if got, _ := classify(c.in); got != c.want {
			t.Errorf("classify(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
```

Fixtures live in `internal/service/receipt/testdata/*.txt` (one OCR transcript
per file); `loadFixture`/`parse` helpers are in `parse_test.go`.
```
