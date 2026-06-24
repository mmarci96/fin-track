package router

import (
	"github.com/gin-gonic/gin"
	"github.com/mmarci96/fin-track/internal/config"
	"github.com/mmarci96/fin-track/internal/handler"
	"github.com/mmarci96/fin-track/internal/httpx"
	"github.com/mmarci96/fin-track/internal/repository"
	ollamapkg "github.com/mmarci96/fin-track/internal/service/ollama"
)

func SetupRouter(
	db *repository.Database,
	cfg *config.AppConfig,
	ollama *ollamapkg.Service,
) *gin.Engine {
	r := gin.New()

	// Order matters: request id first so every later log line is correlated,
	// then recovery so panics are logged with that context, access log, and
	// finally user resolution.
	r.Use(httpx.RequestID())
	r.Use(httpx.Recovery())
	r.Use(httpx.AccessLog())
	r.Use(httpx.UserID(cfg.DefaultUserID, cfg.RequireUserID))

	extractor := ollamapkg.NewItemExtractor(ollama, cfg.OllamaModel)
	imageHandler := handler.NewImageHandler(db, extractor, cfg.ImageStoreDir)
	receiptHandler := handler.NewReceiptHandler(db, ollama)
	merchantHandler := handler.NewMerchantHandler(db)
	categoryHandler := handler.NewCategoryHandler(db, ollama)

	api := r.Group("/api")

	receipts := api.Group("/receipts")
	{
		receipts.POST("/:id/categorize", receiptHandler.CategorizeReceiptItems)
		receipts.POST("/image", imageHandler.OCR)
		// Developer upload: same pipeline, but persists the image + transcript
		// for side-by-side review and recognition analysis.
		receipts.POST("/image/debug", imageHandler.OCRDebug)
		receipts.POST("", receiptHandler.Create)
		receipts.GET("", receiptHandler.List)
		receipts.GET("/:id", receiptHandler.Get)
		receipts.PUT("/:id", receiptHandler.Update)
		receipts.DELETE("/:id", receiptHandler.Delete)
	}

	merchants := api.Group("/merchants")
	{
		merchants.POST("", merchantHandler.Create)
		merchants.GET("", merchantHandler.List)
		merchants.GET("/:id", merchantHandler.Get)
		merchants.PUT("/:id", merchantHandler.Update)
		merchants.DELETE("/:id", merchantHandler.Delete)
	}

	categories := api.Group("/categories")
	{
		categories.POST("", categoryHandler.Create)
		categories.GET("", categoryHandler.List)
		categories.GET("/:id", categoryHandler.Get)
		categories.PUT("/:id", categoryHandler.Update)
		categories.DELETE("/:id", categoryHandler.Delete)
	}

	// Persisted debug uploads: the original image + its transcript/parse, for
	// the developer side-by-side viewer and the recognition dataset.
	images := api.Group("/receipt-images")
	{
		images.GET("", imageHandler.ListImages)
		images.GET("/:id", imageHandler.GetImage)
		images.GET("/:id/meta", imageHandler.GetImageMeta)
		images.PUT("/:id/clean", imageHandler.UpdateImageClean)
	}

	products := api.Group("/products")
	{
		products.PUT("/:id/categories/ai", categoryHandler.CategorizeProductById)
		products.POST("/:id/categories", categoryHandler.Assign)
		products.DELETE("/:id/categories/:cid", categoryHandler.Unassign)
	}

	return r
}
