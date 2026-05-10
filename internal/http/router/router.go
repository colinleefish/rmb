package router

import (
	"github.com/colinleefish/mypast/internal/http/handler"
	"github.com/gin-gonic/gin"
)

func New(
	healthHandler *handler.HealthHandler,
	sessionUploadHandler *handler.SessionUploadHandler,
) *gin.Engine {
	r := gin.Default()

	r.GET("/healthz", healthHandler.Get)
	r.POST("/api/v1/sessions/:session_id/upload", sessionUploadHandler.Upload)

	return r
}
