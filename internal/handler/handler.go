package handler

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/mmarci96/fin-track/internal/repository"
	"github.com/mmarci96/fin-track/internal/service"
	"github.com/mmarci96/fin-track/internal/service/ollama"
)

type ImageHandler struct {
	ollamaSvc *ollama.Service
	db        *repository.Database
}

func NewImageHandler(ollama *ollama.Service, db *repository.Database) *ImageHandler {
	return &ImageHandler{ollamaSvc: ollama, db: db}
}
func (h *ImageHandler) OCR(c *gin.Context) {

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "image required",
		})
		return
	}

	// create temp file
	tmp, err := os.CreateTemp("", "ocr-*.png")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	defer os.Remove(tmp.Name())
	defer tmp.Close()

	// save uploaded file
	if err := c.SaveUploadedFile(
		file,
		tmp.Name(),
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	text, err := service.ParseImageToTxt(
		tmp.Name(),
		true,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	merchants, err := h.db.FindMerchants()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result, err := service.MapReceiptTxt(text, merchants)
	if err != nil {
		if err.Error() == "No Merchant matches from db" {
			c.JSON(http.StatusOK, gin.H{"text": text, "result": result, "warning": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"text":   text,
		"result": result,
	})
}
