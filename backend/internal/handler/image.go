package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mmarci96/fin-track/internal/apperr"
	"github.com/mmarci96/fin-track/internal/httpx"
	"github.com/mmarci96/fin-track/internal/model"
	"github.com/mmarci96/fin-track/internal/repository"
	"github.com/mmarci96/fin-track/internal/service/img"
	"github.com/mmarci96/fin-track/internal/service/receipt"
	"github.com/mmarci96/fin-track/pkg/logger"
)

type ImageHandler struct {
	db        *repository.Database
	extractor receipt.LLMFallback // optional Ollama-backed fallback; may be nil
	storeDir  string              // where debug uploads persist the original image
}

func NewImageHandler(db *repository.Database, extractor receipt.LLMFallback, storeDir string) *ImageHandler {
	return &ImageHandler{db: db, extractor: extractor, storeDir: storeDir}
}

// OCR is the user-facing upload: OCR + parse + store the receipt. The image
// itself is not retained.
func (h *ImageHandler) OCR(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	file, err := c.FormFile("image")
	if err != nil {
		httpx.Respond(c, apperr.BadRequest("image required", errors.Wrap(err, "read form file")))
		return
	}
	log = log.With("filename", file.Filename, "size_bytes", file.Size)
	log.Info("ocr request received")

	// Stage the upload to a temp file for the OCR engine.
	tmp, err := os.CreateTemp("", "ocr-*.png")
	if err != nil {
		httpx.Respond(c, apperr.Internal("could not process upload", errors.Wrap(err, "create temp file")))
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	if err := c.SaveUploadedFile(file, tmp.Name()); err != nil {
		httpx.Respond(c, apperr.Internal("could not process upload", errors.Wrap(err, "save uploaded file")))
		return
	}

	text, result, err := h.ocrAndParse(c.Request.Context(), tmp.Name())
	if err != nil {
		httpx.Respond(c, err)
		return
	}
	// parseImgTxt(text)
	// Nothing usable: ask the user for a clearer photo instead of storing junk.
	if result.Decision == receipt.DecisionReject {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"code":  "RECEIPT_UNREADABLE",
			"error": "could not read the receipt — please retake the photo: lay it flat, good lighting, whole receipt in frame",
			"text":  text,
		})
		return
	}

	stored, err := h.persistReceipt(c.Request.Context(), result)
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result":   stored,
		"decision": result.Decision,
		"warnings": result.Warnings,
		"detected": gin.H{"total": result.Total, "reconciled": result.Reconciled, "merchant_known": result.MerchantKnown},
		"text":     text,
	})
}

func parseImgTxt(txt string) {
	fmt.Printf("text: %s", txt)
	fmt.Print(txt)
	fmt.Println("--------------------------------------------------------------------------")
	fmt.Println(txt)
	fmt.Println("--------------------------------------------------------------------------")
}

// OCRDebug is the developer-facing upload. It runs the same pipeline but ALSO
// persists the original image and a receipt_images record (raw transcript +
// parser output) so the image can later be viewed next to its transcript and
// the corpus can be mined for recognition improvements. The image is saved even
// when the parse is rejected — failures are the most valuable training data.
func (h *ImageHandler) OCRDebug(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())
	userID := httpx.UserIDFromContext(c.Request.Context())

	file, err := c.FormFile("image")
	if err != nil {
		httpx.Respond(c, apperr.BadRequest("image required", errors.Wrap(err, "read form file")))
		return
	}
	log = log.With("filename", file.Filename, "size_bytes", file.Size)
	log.Info("debug ocr request received")

	// Persist the original image to the store, then OCR it in place.
	storedName, fullPath, err := h.saveOriginal(c, file)
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	text, result, err := h.ocrAndParse(c.Request.Context(), fullPath)
	if err != nil {
		// OCR failed outright (e.g. a corrupt file): don't leave the saved image
		// orphaned with no DB row pointing at it.
		_ = os.Remove(fullPath)
		httpx.Respond(c, err)
		return
	}

	// Only create a receipt when the parse is usable; the image record is saved
	// regardless so rejected scans are still captured for analysis.
	var receiptID *int
	var stored *model.Receipt
	if result.Decision != receipt.DecisionReject {
		s, perr := h.persistReceipt(c.Request.Context(), result)
		if perr != nil {
			httpx.Respond(c, perr)
			return
		}
		stored = &s
		receiptID = &s.ID
	}

	parseJSON, _ := json.Marshal(result)
	rec := &model.ReceiptImage{
		ReceiptID:    receiptID,
		UserID:       userID,
		StoredPath:   storedName,
		OriginalName: file.Filename,
		ContentType:  file.Header.Get("Content-Type"),
		OCRText:      text,
		ParseJSON:    parseJSON,
	}
	if err := h.db.CreateReceiptImage(rec); err != nil {
		httpx.Respond(c, apperr.Internal("could not save debug image", err))
		return
	}
	log.Info("debug image stored", "image_id", rec.ID, "decision", string(result.Decision))

	c.JSON(http.StatusOK, gin.H{
		"image_id": rec.ID,
		"result":   stored, // null when rejected
		"decision": result.Decision,
		"warnings": result.Warnings,
		"detected": gin.H{"total": result.Total, "reconciled": result.Reconciled, "merchant_known": result.MerchantKnown},
		"text":     text,
	})
}

// GetImage streams a stored debug image back to the owner.
func (h *ImageHandler) GetImage(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	userID := httpx.UserIDFromContext(c.Request.Context())

	rec, err := h.db.GetReceiptImageByID(id, userID)
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	full := filepath.Join(h.storeDir, rec.StoredPath)
	if _, err := os.Stat(full); err != nil {
		httpx.Respond(c, apperr.NotFound("image file missing", errors.Wrapf(err, "stat %q", full)))
		return
	}
	if rec.ContentType != "" {
		c.Header("Content-Type", rec.ContentType)
	}
	c.File(full)
}

// GetImageMeta returns the transcript + parser output for a debug image so the
// UI can render it next to the picture.
func (h *ImageHandler) GetImageMeta(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	userID := httpx.UserIDFromContext(c.Request.Context())

	rec, err := h.db.GetReceiptImageByID(id, userID)
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            rec.ID,
		"receipt_id":    rec.ReceiptID,
		"original_name": rec.OriginalName,
		"content_type":  rec.ContentType,
		"ocr_text":      rec.OCRText,
		"clean_text":    rec.CleanText,
		"parse":         json.RawMessage(rec.ParseJSON),
		"clean_parse":   json.RawMessage(rec.CleanParseJSON),
		"approved":      rec.Approved,
		"approved_at":   rec.ApprovedAt,
		"created_at":    rec.CreatedAt,
	})
}

// ListImages returns the user's debug uploads (summary rows) newest-first.
func (h *ImageHandler) ListImages(c *gin.Context) {
	userID := httpx.UserIDFromContext(c.Request.Context())
	items, err := h.db.ListReceiptImages(userID)
	if err != nil {
		httpx.Respond(c, apperr.Internal("could not list debug images", err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": items})
}

// UpdateImageClean stores a human-corrected transcript (ground truth) for a
// debug image.
func (h *ImageHandler) UpdateImageClean(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	userID := httpx.UserIDFromContext(c.Request.Context())

	var body struct {
		CleanText string `json:"clean_text"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Respond(c, apperr.BadRequest("clean_text required", errors.Wrap(err, "bind body")))
		return
	}

	// Re-parse the corrected transcript so the viewer can show the structured
	// result live, and persist it as the ground-truth parse (the corpus).
	result, err := h.parseText(c.Request.Context(), body.CleanText)
	if err != nil {
		httpx.Respond(c, err)
		return
	}
	cleanParseJSON, _ := json.Marshal(result)

	if err := h.db.SetReceiptImageClean(id, userID, body.CleanText, cleanParseJSON); err != nil {
		httpx.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "clean_parse": result})
}

// ApproveImage approves (or un-approves) a capture's clean transcript as ground
// truth. The approved state is what the debug viewer surfaces per capture.
func (h *ImageHandler) ApproveImage(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	userID := httpx.UserIDFromContext(ctx)

	var body struct {
		Approved bool `json:"approved"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Respond(c, apperr.BadRequest("approved required", errors.Wrap(err, "bind body")))
		return
	}
	if err := h.db.SetReceiptImageApproved(id, userID, body.Approved); err != nil {
		httpx.Respond(c, err)
		return
	}

	resp := gin.H{"ok": true, "approved": body.Approved}
	// Approving is the ground-truth event: promote the capture into the flywheel
	// by learning a merchant alias from it. Best effort, never blocks approval.
	if body.Approved {
		if alias := h.learnMerchantAlias(ctx, id, userID); alias != nil {
			resp["learned_alias"] = alias
		}
	}
	c.JSON(http.StatusOK, resp)
}

// learnMerchantAlias records the garbled raw-OCR header of an approved capture as
// an alias for the canonical merchant the corrected transcript identifies, so
// future uploads with that header resolve. Returns {alias, merchant} when an
// alias was learned, nil otherwise. Errors are logged, never surfaced — approval
// has already succeeded.
func (h *ImageHandler) learnMerchantAlias(ctx context.Context, id, userID int) gin.H {
	log := logger.FromContext(ctx)

	rec, err := h.db.GetReceiptImageByID(id, userID)
	if err != nil || rec.CleanText == nil || strings.TrimSpace(*rec.CleanText) == "" {
		return nil
	}

	// The canonical merchant comes from re-parsing the corrected transcript.
	cleanRes, err := h.parseText(ctx, *rec.CleanText)
	if err != nil || !cleanRes.MerchantKnown {
		return nil
	}

	// The variant worth learning is the garbled raw-OCR header.
	rawKey := receipt.NormalizeName(firstNonEmptyLine(rec.OCRText))
	if rawKey == "" {
		return nil
	}

	// If the raw OCR already resolves to this merchant (canonical match or an
	// existing alias), there's nothing new to learn.
	if rawRes, perr := h.parseText(ctx, rec.OCRText); perr == nil && rawRes.MerchantKnown &&
		receipt.NormalizeName(rawRes.MerchantName) == receipt.NormalizeName(cleanRes.MerchantName) {
		return nil
	}

	merchantID, err := h.db.ResolveMerchant(cleanRes.MerchantName)
	if err != nil {
		log.Warn("flywheel: resolve merchant failed", "err", err)
		return nil
	}
	if err := h.db.CreateMerchantAlias(merchantID, rawKey, &id); err != nil {
		log.Warn("flywheel: create alias failed", "err", err)
		return nil
	}
	log.Info("flywheel: learned merchant alias", "alias", rawKey, "merchant", cleanRes.MerchantName)
	return gin.H{"alias": rawKey, "merchant": cleanRes.MerchantName}
}

// firstNonEmptyLine returns the first non-blank line of s (trimmed), or "".
func firstNonEmptyLine(s string) string {
	for l := range strings.SplitSeq(s, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			return t
		}
	}
	return ""
}

// ocrAndParse runs OCR over a staged file and parses the text into a graded
// result, scoped to the known merchants and currencies. Shared by both upload
// paths. Returned errors are already apperr-classified.
func (h *ImageHandler) ocrAndParse(ctx context.Context, path string) (string, receipt.Result, error) {
	text, err := img.ParseImageToTxt(path, true)
	if err != nil {
		return "", receipt.Result{}, apperr.Internal("could not read receipt image", err)
	}
	// logger.FromContext(ctx).Debug("ocr extraction complete", "text_len", len(text))
	// logger.FromContext(ctx).Debug("ocr extraction complete", "text", text)

	result, err := h.parseText(ctx, text)
	if err != nil {
		return "", receipt.Result{}, err
	}
	return text, result, nil
}

// parseText loads the reference data (known merchants, currencies, and the
// learned merchant aliases) and runs the parser over arbitrary text. Shared by
// the OCR upload paths and the re-parse of a human-corrected transcript, so both
// benefit from the flywheel's aliases. Returned errors are apperr-classified.
func (h *ImageHandler) parseText(ctx context.Context, text string) (receipt.Result, error) {
	merchants, err := h.db.FindVerifiedMerchants()
	if err != nil {
		return receipt.Result{}, apperr.Internal("could not load merchants", err)
	}
	currencies, err := h.db.GetAllCurrencies()
	if err != nil {
		return receipt.Result{}, apperr.Internal("could not load currencies", err)
	}
	aliases, err := h.db.GetMerchantAliases()
	if err != nil {
		return receipt.Result{}, apperr.Internal("could not load merchant aliases", err)
	}

	result := receipt.NewParser(merchants, currencies, h.extractor).
		WithMerchantAliases(aliases).
		Parse(ctx, text)
	logger.FromContext(ctx).Info("receipt parsed",
		"merchant", result.MerchantName,
		"decision", string(result.Decision),
		"items", len(result.Items),
		"total", result.Total,
		"reconciled", result.Reconciled,
		"confidence", result.Confidence,
		"warnings", result.Warnings,
	)
	return result, nil
}

// persistReceipt resolves the merchant and stores the parsed receipt for the
// acting user.
func (h *ImageHandler) persistReceipt(ctx context.Context, result receipt.Result) (model.Receipt, error) {
	// Only an identified (verified) merchant is persisted; unknown headers collapse
	// onto a single UNKNOWN sentinel rather than minting a junk merchant row per
	// garbled OCR header (the old self-poisoning bug).
	merchantName := "UNKNOWN"
	if n := strings.TrimSpace(result.MerchantName); n != "" {
		merchantName = n
	}
	merchantID, err := h.db.ResolveMerchant(merchantName)
	if err != nil {
		return model.Receipt{}, apperr.Internal("could not save receipt", err)
	}

	stored := toModelReceipt(result, merchantID, httpx.UserIDFromContext(ctx))
	if err := h.db.CreateReceipt(&stored); err != nil {
		return model.Receipt{}, apperr.Internal("could not save receipt", err)
	}
	logger.FromContext(ctx).Info("receipt stored", "receipt_id", stored.ID)
	return stored, nil
}

// saveOriginal copies an uploaded file into the image store under a random name,
// returning the stored (relative) name and the absolute path. The extension is
// preserved so the file is served with a sensible type.
func (h *ImageHandler) saveOriginal(c *gin.Context, file *multipart.FileHeader) (string, string, error) {
	if err := os.MkdirAll(h.storeDir, 0o755); err != nil {
		return "", "", apperr.Internal("could not save image", errors.Wrapf(err, "mkdir %q", h.storeDir))
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = ".img"
	}
	name := uuid.NewString() + ext
	full := filepath.Join(h.storeDir, name)
	if err := c.SaveUploadedFile(file, full); err != nil {
		return "", "", apperr.Internal("could not save image", errors.Wrapf(err, "save %q", full))
	}
	return name, full, nil
}

// pathID parses the :id path param, responding with 400 on failure.
func pathID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.Respond(c, apperr.BadRequest("invalid id", errors.Wrapf(err, "parse id %q", c.Param("id"))))
		return 0, false
	}
	return id, true
}

// toModelReceipt maps a parsed result into the persistence model. total_amount
// uses the printed total when available, otherwise the sum of items.
func toModelReceipt(r receipt.Result, merchantID, userID int) model.Receipt {
	total := r.Total
	if total == 0 {
		total = r.ComputedTotal
	}

	products := make([]model.Product, 0, len(r.Items))
	for _, it := range r.Items {
		products = append(products, model.Product{Name: it.Name, Price: it.Price})
	}

	return model.Receipt{
		UserID:      userID,
		MerchantID:  merchantID,
		Merchant:    model.Merchant{ID: merchantID, Name: r.MerchantName},
		Products:    products,
		TotalAmount: total,
		Currency:    r.Currency,
	}
}
