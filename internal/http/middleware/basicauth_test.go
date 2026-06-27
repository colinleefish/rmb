package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/colinleefish/rmb/internal/config"
	"github.com/gin-gonic/gin"
)

func TestBasicAuthDisabledWhenEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mw, err := BasicAuth(config.AuthConfig{})
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(mw)
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestBasicAuthRequiresBothCredentials(t *testing.T) {
	_, err := BasicAuth(config.AuthConfig{Username: "u"})
	if err == nil {
		t.Fatal("expected error when password missing")
	}
}

func TestBasicAuthRejectsBadCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mw, err := BasicAuth(config.AuthConfig{Username: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(mw)
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
