package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mmarci96/fin-track/internal/apperr"
	"github.com/mmarci96/fin-track/internal/httpx"
	"github.com/mmarci96/fin-track/internal/model"
	"github.com/mmarci96/fin-track/internal/repository"
	"github.com/mmarci96/fin-track/internal/service/ollama"
)

type ProductHandler struct {
	db     *repository.Database
	ollama *ollama.Service
}

func NewProductHandler(db *repository.Database, ollama *ollama.Service) *ProductHandler {
	return &ProductHandler{db: db, ollama: ollama}
}

func (h *ProductHandler) Create(c *gin.Context) {
	var req model.Product
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Respond(c, apperr.BadRequest("invalid request body", err))
		return
	}

	product, err := h.db.CreateProduct(req)
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"result": product})
}

func (h *ProductHandler) Get(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		httpx.Respond(c, err)
		return
	}
	product, err := h.db.GetProductByID(id)
	if err != nil {
		httpx.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": product})
}

func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		httpx.Respond(c, err)
		return
	}
	err = h.db.DeleteProduct(id)
	if err != nil {
		httpx.Respond(c, err)
		return
	}
	c.JSON(http.StatusNoContent, gin.H{})
}

func (h *ProductHandler) Update(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	var req model.Product
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Respond(c, apperr.BadRequest("invalid request body", err))
		return
	}

	req.ID = id
	product, err := h.db.UpdateProduct(id, req)
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": product})
}

func (h *ProductHandler) Remove(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		httpx.Respond(c, err)
		return
	}
	err = h.db.DeleteProduct(id)
	if err != nil {
		httpx.Respond(c, err)
		return
	}
	c.JSON(http.StatusNoContent, gin.H{})
}
