# Receipt recognition roadmap

How fin-track turns "scan a receipt" from a one-shot OCR guess into a system that
gets measurably better as the database grows. The guiding idea: **every scan we
keep — image + raw transcript + (corrected) clean transcript — is training data**.
The more confirmed receipts we have, the more we can canonicalize product names,
auto-categorize, and catch errors/anomalies with cheap rules instead of the LLM.

This is a living plan, not a spec. Phases are ordered by dependency and value;
each phase is independently shippable.

## Where we are today

Pipeline (see `internal/service/receipt`):

```
OCR (tesseract) → normalize → detectMerchant (scoped to a known-merchants list)
  → classify/extract lines → reconcile vs printed total
  → decide (accepted / best_effort / reject) → optional LLM fallback
```

Key facts the roadmap builds on:

- **Known-merchants pattern already works.** `merchants.normalized_name` is a
  canonical key; OCR variants collapse onto it. We replicate this for products.
- **Graded decisions + warnings** already exist (`reconcile`, `confidence`,
  `WarnTotalMismatch`, …) — the hooks for anomaly flags are in place.
- **The dataset capture is built** (Phase 0): the debug upload endpoint
  `POST /api/receipts/image/debug` persists the original image plus `ocr_text`
  and `parse_json` into `receipt_images`, and the dev-only `/debug` UI shows the
  image next to its transcript with a **clean transcript** editor
  (`PUT /api/receipt-images/:id/clean`). That clean text is the ground truth
  everything below learns from.

## Phase 0 — Capture the corpus ✅ (shipped)

- `receipt_images` table: `stored_path`, `ocr_text`, `clean_text`, `parse_json`,
  nullable `receipt_id` (so rejected scans and deleted receipts still leave their
  training data behind).
- Images on a docker volume (`IMAGE_STORE_DIR`); DB keeps only the path.
- Dev viewer for side-by-side review + correction.

**Next step to make it useful:** scan a batch of real receipts through the debug
endpoint and correct the transcripts. ~50–100 corrected examples is enough to
start Phase 1 and to make the Phase 4 scoring harness meaningful.

## Phase 1 — Known-products catalog (highest value)

Mirror the known-merchants mechanism for line items.

New table (sketch):

```sql
CREATE TABLE known_products (
    id              SERIAL PRIMARY KEY,
    normalized_name TEXT NOT NULL UNIQUE,   -- matches receipt.NormalizeName
    canonical_name  TEXT NOT NULL,          -- clean display name
    category_id     INTEGER REFERENCES categories(id),
    price_min       INTEGER,                -- per-currency stats, minor units
    price_median    INTEGER,
    price_max       INTEGER,
    currency_id     INTEGER REFERENCES currencies(id),
    seen_count      INTEGER NOT NULL DEFAULT 0,
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
);
```

Built by aggregating **confirmed** products (accepted receipts + corrected debug
uploads), not raw guesses — garbage in, garbage out.

At parse time, after `extractItems`, match each item against `known_products`
(normalized exact match first, then bounded edit-distance / trigram similarity):

1. **Canonicalize the name** — fix OCR drift ("Maretti" vs "Maretu", leading
   column codes) by snapping to `canonical_name` on a confident match.
2. **Auto-assign category** — skip the LLM categorize call when we already know
   the product's category. Cheaper and more consistent.
3. **Price sanity check** — flag items whose price falls far outside
   `[price_min, price_max]` (or N×median) as a new warning, e.g.
   `WarnPriceAnomaly`. This catches merged-digit OCR errors ("9399" for "399").

Payoff scales with the DB: more confirmed receipts → bigger catalog → more items
recognized without the LLM and more anomalies caught. This is the
"high-value as our DB grows" goal.

## Phase 2 — Learn OCR error rules from raw↔clean pairs

Once we have raw `ocr_text` paired with human `clean_text`, mine the diffs for
**systematic** OCR confusions and encode the recurring ones as targeted rules in
`normalize.go` / `extract.go`:

- character confusions: `0↔O`, `5↔S`, `1↔I`, `8↔B`;
- digit merges in the price column (the layout reconstruction in
  `service/img/layout.go` is the first defense; add a post-check);
- merchant-specific quirks (Aldi column codes, OTP unit-price lines) — extend the
  existing per-layout handling.

Rule of thumb: only promote a fix to a rule when the corpus shows it happening
repeatedly across receipts; one-offs stay one-offs. Each new rule must be backed
by a fixture (Phase 4) so it can't silently regress other receipts.

## Phase 3 — Anomaly detection beyond single items

With per-merchant and per-product history, add cross-checks:

- **Total vs sum** — already done (`reconcile`); keep surfacing as a warning.
- **Per-merchant distributions** — typical item count, typical line price range;
  flag a "groceries" receipt with one €900 line.
- **Duplicate line detection** — same item twice in a row often means an OCR
  double-read vs a genuine 2× purchase; flag for review.
- **Currency/decimal sanity** — HUF has 0 decimals, EUR 2; flag implausible
  magnitudes.

These feed the `Decision`/`Warnings` model already in `types.go`.

## Phase 4 — Scoring harness (makes "learn from mistakes" measurable)

Turn the corpus into a regression test. Replay every stored image through the
current pipeline and compare against its clean transcript:

- **item precision/recall** (did we find the right line items?),
- **name accuracy** after canonicalization,
- **total accuracy** and reconcile rate,
- **category accuracy** where labels exist.

Extends the existing golden-fixture tests (`parse_test.go`, `otp_test.go`): those
are hand-written fixtures; this is the same idea driven by the growing corpus.
Every heuristic or rule change (Phases 1–3) is then scored against the whole
corpus before merge — no change "fixes one receipt and breaks five" unnoticed.

## Phase 5 — Feedback loop & LLM tuning

- Confirmed corrections continuously update `known_products` (names, categories,
  price stats) and the fixture corpus.
- Use the best raw↔clean examples as few-shot context for the LLM fallback
  prompt, or to fine-tune, so the fallback also improves over time.
- Optionally, route low-confidence parses to the debug/correction queue
  automatically so the humans-in-the-loop effort targets exactly the receipts
  the system is worst at.

## Sequencing summary

| Phase | What | Depends on | Status |
|------|------|-----------|--------|
| 0 | Capture image + raw + clean transcript | — | ✅ shipped |
| 1 | `known_products` catalog: canonicalize, auto-category, price anomaly | corpus of confirmed items | next |
| 2 | OCR error rules mined from raw↔clean | corrected transcripts | after 1 |
| 3 | Cross-item / per-merchant anomaly checks | history | after 1 |
| 4 | Scoring harness over the corpus | corpus | parallel with 1 |
| 5 | Feedback loop + LLM tuning | 1–4 | later |

The single most valuable next action is mundane: **collect and correct a few
dozen real receipts** through the debug tool. Everything downstream is only as
good as that ground-truth corpus.
