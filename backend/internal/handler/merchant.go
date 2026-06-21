package handler

import (
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"
	"github.com/mmarci96/fin-track/internal/apperr"
	"github.com/mmarci96/fin-track/internal/httpx"
	"github.com/mmarci96/fin-track/internal/repository"
)

// MerchantHandler serves CRUD for merchants, which are global reference data.
type MerchantHandler struct {
	db *repository.Database
}

func NewMerchantHandler(db *repository.Database) *MerchantHandler {
	return &MerchantHandler{db: db}
}

type merchantRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *MerchantHandler) Create(c *gin.Context) {
	var req merchantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Respond(c, apperr.BadRequest("invalid request body", errors.Wrap(err, "bind merchant")))
		return
	}

	merchant, err := h.db.CreateMerchant(req.Name)
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"result": merchant})
}

func (h *MerchantHandler) List(c *gin.Context) {
	merchants, err := h.db.GetAllMerchants()
	if err != nil {
		httpx.Respond(c, apperr.Internal("could not list merchants", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": merchants})
}

func (h *MerchantHandler) Get(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	merchant, err := h.db.GetMerchantByID(id)
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": merchant})
}

func (h *MerchantHandler) Update(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	var req merchantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Respond(c, apperr.BadRequest("invalid request body", errors.Wrap(err, "bind merchant")))
		return
	}

	merchant, err := h.db.UpdateMerchant(id, req.Name)
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": merchant})
}

func (h *MerchantHandler) Delete(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	if err := h.db.DeleteMerchant(id); err != nil {
		httpx.Respond(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
