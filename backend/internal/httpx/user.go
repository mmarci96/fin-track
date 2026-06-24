package httpx

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mmarci96/fin-track/internal/apperr"
	"github.com/mmarci96/fin-track/pkg/logger"
)

// UserIDHeader identifies the acting user. It is set by the trusted edge: the
// Traefik `traefikauth` plugin strips any client-supplied value and injects the
// id from a verified auth-service JWT. See the auth-service module.
const UserIDHeader = "X-User-ID"

type userIDKey struct{}

// UserID resolves the acting user for the request from the X-User-ID header.
//
// When require is false (local dev hitting the backend directly, without the
// edge), a missing header falls back to defaultID so development stays
// frictionless. When require is true (UAT/prod behind Traefik), a missing
// header means the request did not pass the auth edge and is rejected — the
// backend must never invent an identity in that case.
//
// This is the seam for authentication: only this middleware changes.
func UserID(defaultID int, require bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := defaultID
		raw := c.GetHeader(UserIDHeader)
		switch {
		case raw != "":
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed <= 0 {
				Respond(c, apperr.BadRequest("invalid X-User-ID header", nil))
				return
			}
			id = parsed
		case require:
			Respond(c, apperr.Unauthorized("missing X-User-ID header", nil))
			return
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
