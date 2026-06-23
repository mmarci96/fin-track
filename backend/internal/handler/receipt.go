package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"
	"github.com/mmarci96/fin-track/internal/apperr"
	"github.com/mmarci96/fin-track/internal/httpx"
	"github.com/mmarci96/fin-track/internal/model"
	"github.com/mmarci96/fin-track/internal/repository"
	"github.com/mmarci96/fin-track/internal/service/ollama"
)

// ReceiptHandler serves the non-OCR receipt CRUD endpoints. All operations are
// scoped to the acting user from the request context.
type ReceiptHandler struct {
	db *repository.Database
	ollama *ollama.Service
}

func NewReceiptHandler(db *repository.Database, ollama *ollama.Service) *ReceiptHandler {
	return &ReceiptHandler{db: db, ollama: ollama}
}

type productInput struct {
	Name  string `json:"name" binding:"required"`
	Price int    `json:"price"`
}

type receiptCreateRequest struct {
	MerchantID  int            `json:"merchant_id" binding:"required"`
	TotalAmount int            `json:"total_amount"`
	Currency    string         `json:"currency"`
	Products    []productInput `json:"products"`
}

type receiptUpdateRequest struct {
	TotalAmount int            `json:"total_amount"`
	Currency    string         `json:"currency"`
	Products    []productInput `json:"products"`
}

func toProducts(in []productInput) []model.Product {
	products := make([]model.Product, 0, len(in))
	for _, p := range in {
		products = append(products, model.Product{Name: p.Name, Price: p.Price})
	}
	return products
}

// parseIDParam reads a positive integer path parameter or returns a 400.
func parseIDParam(c *gin.Context, name string) (int, error) {
	id, err := strconv.Atoi(c.Param(name))
	if err != nil || id <= 0 {
		return 0, apperr.BadRequest("invalid "+name, errors.Wrapf(err, "parse %s", name))
	}
	return id, nil
}

func (h *ReceiptHandler) CategorizeReceiptItems(c *gin.Context) {
		id, err := parseIDParam(c, "id")
	if err != nil {
		httpx.Respond(c, err)
		return
	}
	userID := httpx.UserIDFromContext(c.Request.Context())

	receipt, err := h.db.GetReceiptByID(id, userID)
	if err != nil {
		httpx.Respond(c, err)
		return
	}
	
	items := receipt.Products
	
	availableCategories, err := h.db.GetAllCategories()
	if err != nil {
		httpx.Respond(c, err)
		return
	}
	categoryNames := make([]string, len(availableCategories))
	for i, category := range availableCategories {
		categoryNames[i] = category.Name
	}
	reuslts := make([]model.GenerateResponse, 0, len(items))
	for _, item := range items {
		prompt := "Ezek kozul a kategoriak kozul: " + strings.Join(categoryNames, ", ") + ". -> Valaszd ki melyik kategoriakba tartozik (tobb is valaszthato): " + item.Name
		res, err := h.ollama.Generate(c, model.GenerateRequest{Model: "qwen3:1.7b", Prompt: prompt, Think: false})
		if err != nil {
			httpx.Respond(c,err)
		}
		reuslts = append(reuslts, *res)
	}

	if err != nil {
		httpx.Respond(c,err)
	}
	c.JSON(http.StatusOK, gin.H{"result": reuslts})
}

func (h *ReceiptHandler) Create(c *gin.Context) {
	var req receiptCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Respond(c, apperr.BadRequest("invalid request body", errors.Wrap(err, "bind receipt")))
		return
	}

	receipt := model.Receipt{
		UserID:      httpx.UserIDFromContext(c.Request.Context()),
		MerchantID:  req.MerchantID,
		TotalAmount: req.TotalAmount,
		Currency:    model.Currency{Code: req.Currency},
		Products:    toProducts(req.Products),
	}

	if err := h.db.CreateReceipt(&receipt); err != nil {
		httpx.Respond(c, apperr.Internal("could not create receipt", err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"result": receipt})
}

func (h *ReceiptHandler) List(c *gin.Context) {
	userID := httpx.UserIDFromContext(c.Request.Context())

	receipts, err := h.db.GetAllReceipts(userID)
	if err != nil {
		httpx.Respond(c, apperr.Internal("could not list receipts", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": receipts})
}

func (h *ReceiptHandler) Get(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		httpx.Respond(c, err)
		return
	}
	userID := httpx.UserIDFromContext(c.Request.Context())

	receipt, err := h.db.GetReceiptByID(id, userID)
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": receipt})
}

func (h *ReceiptHandler) Update(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	var req receiptUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Respond(c, apperr.BadRequest("invalid request body", errors.Wrap(err, "bind receipt")))
		return
	}
	userID := httpx.UserIDFromContext(c.Request.Context())

	receipt := model.Receipt{
		TotalAmount: req.TotalAmount,
		Currency:    model.Currency{Code: req.Currency},
		Products:    toProducts(req.Products),
	}

	if err := h.db.UpdateReceiptByID(id, userID, &receipt); err != nil {
		httpx.Respond(c, err)
		return
	}

	updated, err := h.db.GetReceiptByID(id, userID)
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": updated})
}

func (h *ReceiptHandler) Delete(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		httpx.Respond(c, err)
		return
	}
	userID := httpx.UserIDFromContext(c.Request.Context())

	if err := h.db.RemoveReceiptByID(id, userID); err != nil {
		httpx.Respond(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
