package router

import (
	"io/fs"
	"net/http"

	"github.com/colinleefish/mypast/internal/config"
	"github.com/colinleefish/mypast/internal/http/handler"
	"github.com/colinleefish/mypast/internal/http/middleware"
	"github.com/colinleefish/mypast/internal/http/static"
	"github.com/gin-gonic/gin"
)

func New(
	cfg config.Config,
	healthHandler *handler.HealthHandler,
	sessionUploadHandler *handler.SessionUploadHandler,
	browseHandler *handler.BrowseHandler,
	recallHandler *handler.RecallHandler,
	inspectHandler *handler.InspectHandler,
	assertionHandler *handler.AssertionHandler,
) (*gin.Engine, error) {
	r := gin.Default()

	r.GET("/healthz", healthHandler.Get)

	authMW, err := middleware.BasicAuth(cfg.Auth)
	if err != nil {
		return nil, err
	}

	protected := r.Group("/")
	protected.Use(authMW)

	webFS, err := fs.Sub(static.Web, "web")
	if err != nil {
		return nil, err
	}
	protected.StaticFS("/ui", http.FS(webFS))
	protected.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/ui/")
	})

	protected.POST("/api/v1/sessions/:session_id/upload", sessionUploadHandler.Upload)

	api := protected.Group("/api/v1/browse")
	{
		api.GET("/overview", browseHandler.Overview)
		api.GET("/sessions", browseHandler.ListSessions)
		api.GET("/sessions/:session_key", browseHandler.GetSession)
		api.GET("/atoms", browseHandler.ListAtoms)
		api.GET("/scenes", browseHandler.ListScenes)
		api.GET("/memories", browseHandler.ListMemories)
		api.GET("/pipeline-state", browseHandler.ListPipelineStates)
		api.GET("/tasks", browseHandler.ListTasks)
	}

	if recallHandler != nil {
		protected.GET("/api/v1/find", recallHandler.Find)
		protected.GET("/api/v1/search", recallHandler.Search)
	}

	if inspectHandler != nil {
		protected.GET("/api/v1/inspect/cat", inspectHandler.Cat)
		protected.GET("/api/v1/inspect/tree", inspectHandler.Tree)
		protected.GET("/api/v1/inspect/meta", inspectHandler.Meta)
	}

	if assertionHandler != nil {
		protected.POST("/api/v1/assertions", assertionHandler.Create)
	}

	return r, nil
}
