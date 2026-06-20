package handler

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/mmarci96/fin-track/internal/service"
)

type ImageHandler struct{}

func NewImageHandler() *ImageHandler {
	return &ImageHandler{}
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"text": text,
	})
}

// func (h *ImageHandler) ParseImgToTxt(c *gin.Context) {
// 	var req model.GenerateRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error":   "Invalid input format",
// 			"details": err.Error(),
// 		})
// 		return
// 	}
//
// 	service.ParseImageToTxt()
// }
