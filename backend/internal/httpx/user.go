package httpx

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mmarci96/fin-track/internal/apperr"
	"github.com/mmarci96/fin-track/pkg/logger"
)

// UserIDHeader optionally identifies the uploader until real auth exists.
const UserIDHeader = "X-User-ID"

type userIDKey struct{}

// UserID resolves the acting user for the request: an inbound X-User-ID header
// if present and valid, otherwise the configured default. The id is stored in
// the request context and added to the scoped logger.
//
// This is the seam for real authentication later — only this middleware changes.
func UserID(defaultID int) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := defaultID
		if raw := c.GetHeader(UserIDHeader); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed <= 0 {
				Respond(c, apperr.BadRequest("invalid X-User-ID header", nil))
				return
			}
			id = parsed
		}

		ctx := context.WithValue(c.Request.Context(), userIDKey{}, id)
		ctx = logger.ToContext(ctx, logger.FromContext(ctx).With("user_id", id))
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// UserIDFromContext returns the acting user id, or 0 if none was set (which
// should not happen when the UserID middleware is installed).
func UserIDFromContext(ctx context.Context) int {
	if id, ok := ctx.Value(userIDKey{}).(int); ok {
		return id
	}
	return 0
}
