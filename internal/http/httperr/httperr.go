// Package httperr maps handler errors to safe HTTP JSON responses.
package httperr

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/colinleefish/rmb/internal/service/correction"
	"github.com/colinleefish/rmb/internal/service/session"
	"github.com/colinleefish/rmb/internal/uri"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// JSON writes a client-facing error payload.
func JSON(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}

// Write logs unexpected failures and returns a safe JSON error body.
// Domain validation errors and not-found cases pass through a public message.
func Write(c *gin.Context, err error) {
	if err == nil {
		return
	}

	status, msg := classify(err)
	if status >= http.StatusInternalServerError {
		log.Printf("%s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	}
	JSON(c, status, msg)
}

func classify(err error) (int, string) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return http.StatusNotFound, "not found"
	case errors.Is(err, uri.ErrInvalidURI):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, correction.ErrInvalidInput), errors.Is(err, session.ErrInvalidUploadInput):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, correction.ErrNotFound):
		return http.StatusNotFound, err.Error()
	case isPublicMessage(err.Error()):
		return http.StatusBadRequest, err.Error()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

func isPublicMessage(msg string) bool {
	lower := strings.ToLower(msg)
	for _, sub := range []string{
		"sql", "pq:", "gorm", "postgres", "connection refused",
		"dial tcp", "timeout", "internal", "marshal", "unmarshal",
	} {
		if strings.Contains(lower, sub) {
			return false
		}
	}
	return msg != ""
}
