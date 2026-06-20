package router

import (
	"github.com/gin-gonic/gin"
	"github.com/mmarci96/fin-track/internal/config"
	"github.com/mmarci96/fin-track/internal/handler"
	"github.com/mmarci96/fin-track/internal/repository"
)

func SetupRouter(db *repository.Database, cfg *config.AppConfig) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(ErrorHandler())

	r.POST("/api/receipts/image", handler.NewImageHandler().OCR)

	return r
}
