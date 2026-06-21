package handler

import (
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"
	"github.com/mmarci96/fin-track/internal/apperr"
	"github.com/mmarci96/fin-track/internal/httpx"
	"github.com/mmarci96/fin-track/internal/repository"
)

// CategoryHandler serves CRUD for categories plus product assignment.
// Categories are global reference data.
type CategoryHandler struct {
	db *repository.Database
}

func NewCategoryHandler(db *repository.Database) *CategoryHandler {
	return &CategoryHandler{db: db}
}

type categoryRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *CategoryHandler) Create(c *gin.Context) {
	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Respond(c, apperr.BadRequest("invalid request body", errors.Wrap(err, "bind category")))
		return
	}

	cat, err := h.db.CreateCategory(req.Name)
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"result": cat})
}

func (h *CategoryHandler) List(c *gin.Context) {
	categories, err := h.db.GetAllCategories()
	if err != nil {
		httpx.Respond(c, apperr.Internal("could not list categories", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": categories})
}

func (h *CategoryHandler) Get(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	cat, err := h.db.GetCategoryByID(id)
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": cat})
}

func (h *CategoryHandler) Update(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Respond(c, apperr.BadRequest("invalid request body", errors.Wrap(err, "bind category")))
		return
	}

	cat, err := h.db.UpdateCategory(id, req.Name)
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": cat})
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	if err := h.db.DeleteCategory(id); err != nil {
		httpx.Respond(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Assign links a category to a product. Route: POST /products/:id/categories
func (h *CategoryHandler) Assign(c *gin.Context) {
	productID, err := parseIDParam(c, "id")
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	var req struct {
		CategoryID int `json:"category_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Respond(c, apperr.BadRequest("invalid request body", errors.Wrap(err, "bind assignment")))
		return
	}

	if err := h.db.AssignCategoryToProduct(productID, req.CategoryID); err != nil {
		httpx.Respond(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Unassign removes a category from a product.
// Route: DELETE /products/:id/categories/:cid
func (h *CategoryHandler) Unassign(c *gin.Context) {
	productID, err := parseIDParam(c, "id")
	if err != nil {
		httpx.Respond(c, err)
		return
	}
	categoryID, err := parseIDParam(c, "cid")
	if err != nil {
		httpx.Respond(c, err)
		return
	}

	if err := h.db.UnassignCategoryFromProduct(productID, categoryID); err != nil {
		httpx.Respond(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
