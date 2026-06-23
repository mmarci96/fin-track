package handler

import (
	"net/http"
	"os"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"
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
}

func NewImageHandler(db *repository.Database, extractor receipt.LLMFallback) *ImageHandler {
	return &ImageHandler{db: db, extractor: extractor}
}

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

	// Stage 1: OCR.
	text, err := img.ParseImageToTxt(tmp.Name(), true)
	if err != nil {
		httpx.Respond(c, apperr.Internal("could not read receipt image", err))
		return
	}
	log.Debug("ocr extraction complete", "text_len", len(text))

	// Stage 2: parse (heuristics + optional LLM fallback), scoped to known merchants.
	merchants, err := h.db.FindMerchants()
	if err != nil {
		httpx.Respond(c, apperr.Internal("could not load merchants", err))
		return
	}

	currencies, err := h.db.GetAllCurrencies()
	if err != nil {
		httpx.Respond(c, apperr.Internal("could not load currencies", err))
		return
	}

	result := receipt.NewParser(merchants, currencies, h.extractor).Parse(c.Request.Context(), text)
	log = log.With("merchant", result.MerchantName, "decision", string(result.Decision))
	log.Info("receipt parsed",
		"items", len(result.Items),
		"total", result.Total,
		"reconciled", result.Reconciled,
		"confidence", result.Confidence,
		"warnings", result.Warnings,
	)

	// Nothing usable: ask the user for a clearer photo instead of storing junk.
	if result.Decision == receipt.DecisionReject {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"code":  "RECEIPT_UNREADABLE",
			"error": "could not read the receipt — please retake the photo: lay it flat, good lighting, whole receipt in frame",
			"text":  text,
		})
		return
	}

	// Stage 3: persist, owned by the acting user, merchant de-duped by normalized name.
	merchantName := strings.TrimSpace(result.MerchantName)
	if merchantName == "" {
		merchantName = "UNKNOWN"
	}
	merchantID, err := h.db.ResolveMerchant(merchantName)
	if err != nil {
		httpx.Respond(c, apperr.Internal("could not save receipt", err))
		return
	}

	stored := toModelReceipt(result, merchantID, httpx.UserIDFromContext(c.Request.Context()))
	if err := h.db.CreateReceipt(&stored); err != nil {
		httpx.Respond(c, apperr.Internal("could not save receipt", err))
		return
	}
	log.Info("receipt stored", "receipt_id", stored.ID)

	c.JSON(http.StatusOK, gin.H{
		"result":   stored,
		"decision": result.Decision,
		"warnings": result.Warnings,
		"detected": gin.H{"total": result.Total, "reconciled": result.Reconciled, "merchant_known": result.MerchantKnown},
		"text":     text,
	})
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
