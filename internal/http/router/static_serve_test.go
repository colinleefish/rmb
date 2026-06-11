package router

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/colinleefish/mypast/internal/http/static"
	"github.com/gin-gonic/gin"
)

// TestUIDirectRouteServing guards against the trailing-slash regression: the
// Next.js export uses trailingSlash:true, so directly visiting a sub-route
// (e.g. /ui/memories/) must serve memories/index.html rather than 404.
func TestUIDirectRouteServing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	webFS, err := fs.Sub(static.Web, "web")
	if err != nil {
		t.Fatalf("fs.Sub web: %v", err)
	}

	r := gin.New()
	uiFileServer := http.StripPrefix("/ui", http.FileServer(http.FS(webFS)))
	r.GET("/ui/*filepath", gin.WrapH(uiFileServer))
	r.HEAD("/ui/*filepath", gin.WrapH(uiFileServer))

	cases := []struct {
		path string
		want int
	}{
		{"/ui/", http.StatusOK},
		{"/ui/memories/", http.StatusOK},
		{"/ui/atoms/", http.StatusOK},
		{"/ui/sessions/", http.StatusOK},
		{"/ui/nonexistent/", http.StatusNotFound},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Errorf("GET %s = %d, want %d", tc.path, w.Code, tc.want)
		}
	}
}
