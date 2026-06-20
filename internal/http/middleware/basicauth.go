package middleware

import (
	"fmt"
	"strings"

	"github.com/colinleefish/mem9/internal/config"
	"github.com/gin-gonic/gin"
)

// BasicAuth returns Gin basic-auth middleware when credentials are configured.
// When both username and password are empty, all routes pass through unchanged.
func BasicAuth(cfg config.AuthConfig) (gin.HandlerFunc, error) {
	if !cfg.Enabled() {
		return func(c *gin.Context) { c.Next() }, nil
	}
	if strings.TrimSpace(cfg.Username) == "" || strings.TrimSpace(cfg.Password) == "" {
		return nil, fmt.Errorf("auth: set both USERNAME and PASSWORD (or MEM9_USERNAME and MEM9_PASSWORD)")
	}
	accounts := gin.Accounts{
		cfg.Username: cfg.Password,
	}
	return gin.BasicAuth(accounts), nil
}
