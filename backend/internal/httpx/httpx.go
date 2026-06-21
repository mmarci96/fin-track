// Package httpx holds the HTTP-boundary concerns shared by the router and
// handlers: request-scoped logging, panic recovery, and the single place that
// turns an error into a client response. Keeping it separate from router and
// handler avoids an import cycle.
package httpx

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mmarci96/fin-track/internal/apperr"
	"github.com/mmarci96/fin-track/pkg/logger"
)

// RequestIDHeader is honored on the way in and echoed on the way out, so a
// front proxy or client can correlate logs end to end.
const RequestIDHeader = "X-Request-ID"

// RequestID attaches a request id and a request-scoped logger to the request
// context. Inbound X-Request-ID is reused if present; otherwise one is
// generated.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Writer.Header().Set(RequestIDHeader, id)

		l := logger.With(
			"request_id", id,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
		)

		ctx := logger.ToContext(c.Request.Context(), l)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// AccessLog emits one structured line per completed request, replacing
// gin.Logger so access logs share the format and request id of everything else.
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		logger.FromContext(c.Request.Context()).Info(
			"request completed",
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}

// Recovery logs panics with a stack trace through the structured logger and
// returns a clean 500, replacing both gin.Recovery and the old fmt.Println
// handler.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = apperr.Internal("internal server error", nil)
				} else {
					err = apperr.Internal("internal server error", err)
				}

				logger.FromContext(c.Request.Context()).Error(
					"panic recovered",
					apperr.LogArgs(err)...,
				)

				if !c.Writer.Written() {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
						"error": "internal server error",
						"code":  "INTERNAL",
					})
				}
			}
		}()
		c.Next()
	}
}

// Respond is the single exit point for failed requests. It maps err to a status
// and client-safe message, logs the full detail (with stack) once at the
// boundary, and writes the JSON response.
func Respond(c *gin.Context, err error) {
	status, code, public := apperr.HTTPResponse(err)

	l := logger.FromContext(c.Request.Context())
	if status >= http.StatusInternalServerError {
		l.Error("request failed", apperr.LogArgs(err)...)
	} else {
		l.Warn("request rejected", apperr.LogArgs(err)...)
	}

	c.AbortWithStatusJSON(status, gin.H{
		"error": public,
		"code":  code,
	})
}
