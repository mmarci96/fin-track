package router

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("Recovered from panic:", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Recovered Internal Server Error",
				})
			}
		}()
		c.Next()
	}
}
