package handler

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/colinleefish/rmb/internal/http/httperr"
	"github.com/colinleefish/rmb/internal/service/inspect"
	"github.com/gin-gonic/gin"
)

// InspectHandler exposes cat/tree/meta over HTTP, reusing the inspect service so
// the textual output matches the local CLI byte-for-byte.
type InspectHandler struct {
	svc *inspect.Service
}

func NewInspectHandler(svc *inspect.Service) *InspectHandler {
	return &InspectHandler{svc: svc}
}

func (h *InspectHandler) Cat(c *gin.Context) {
	h.run(c, h.svc.Cat)
}

func (h *InspectHandler) Tree(c *gin.Context) {
	h.run(c, h.svc.Tree)
}

func (h *InspectHandler) Meta(c *gin.Context) {
	h.run(c, h.svc.Meta)
}

func (h *InspectHandler) run(
	c *gin.Context,
	fn func(context.Context, string, io.Writer) error,
) {
	uri := c.Query("uri")
	if uri == "" {
		httperr.JSON(c, http.StatusBadRequest, "uri is required")
		return
	}
	var buf bytes.Buffer
	if err := fn(c.Request.Context(), uri, &buf); err != nil {
		httperr.Write(c, err)
		return
	}
	c.Data(http.StatusOK, "text/plain; charset=utf-8", buf.Bytes())
}
