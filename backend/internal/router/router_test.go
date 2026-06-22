package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mmarci96/fin-track/internal/config"
)

// TestSetupRouterRegisters ensures all routes register without gin panicking on
// conflicting paths (e.g. static /image vs /:id). Handlers are not invoked, so
// nil dependencies are fine.
func TestSetupRouterRegisters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("SetupRouter panicked: %v", r)
		}
	}()

	r := SetupRouter(nil, &config.AppConfig{DefaultUserID: 1}, nil, nil)
	if len(r.Routes()) == 0 {
		t.Fatal("expected routes to be registered")
	}
}
