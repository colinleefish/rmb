package httperr

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/colinleefish/rmb/internal/service/correction"
	"github.com/colinleefish/rmb/internal/service/session"
	"github.com/colinleefish/rmb/internal/uri"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		public string
	}{
		{"not found", gorm.ErrRecordNotFound, http.StatusNotFound, "not found"},
		{"wrapped not found", fmt.Errorf("load: %w", gorm.ErrRecordNotFound), http.StatusNotFound, "not found"},
		{"invalid uri", uri.ErrInvalidURI, http.StatusBadRequest, "invalid rmb uri"},
		{"correction input", correction.ErrInvalidInput, http.StatusBadRequest, "invalid correction input"},
		{"upload input", session.ErrInvalidUploadInput, http.StatusBadRequest, "invalid session upload input"},
		{"correction missing", correction.ErrNotFound, http.StatusNotFound, "correction not found"},
		{"user message", errors.New("session id required"), http.StatusBadRequest, "session id required"},
		{"internal sql", errors.New("sql: connection refused"), http.StatusInternalServerError, "internal server error"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, msg := classify(tc.err)
			if status != tc.status || msg != tc.public {
				t.Fatalf("classify() = (%d, %q), want (%d, %q)", status, msg, tc.status, tc.public)
			}
		})
	}
}

func TestWriteInternalDoesNotLeak(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	Write(c, fmt.Errorf("sql: relation \"secrets\" does not exist"))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
	if body := w.Body.String(); strings.Contains(body, "secrets") || strings.Contains(body, "relation") {
		t.Fatalf("leaked internal error: %s", body)
	}
}
