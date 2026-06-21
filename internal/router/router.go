package router

import (
	"github.com/gin-gonic/gin"
	"github.com/mmarci96/fin-track/internal/config"
	"github.com/mmarci96/fin-track/internal/handler"
	"github.com/mmarci96/fin-track/internal/repository"
	"github.com/mmarci96/fin-track/internal/service/ollama"
)

func SetupRouter(db *repository.Database, cfg *config.AppConfig, ollama *ollama.Service) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(ErrorHandler())

	r.POST("/api/receipts/image", handler.NewImageHandler(ollama, db).OCR)

	return r
}
