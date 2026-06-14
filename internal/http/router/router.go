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
	backfillHandler *handler.BackfillHandler,
	embedHandler *handler.EmbedHandler,
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
	// Serve the embedded Next.js static export directly via http.FileServer
	// instead of gin's StaticFS. The export uses trailingSlash:true, so routes
	// like /ui/memories/ must resolve to memories/index.html. gin's StaticFS
	// rejects these because its existence check (fs.Open("/memories/")) treats
	// the trailing slash as an invalid embed.FS path and returns 404 before the
	// file server runs. http.FileServer handles directory index files natively.
	uiFileServer := http.StripPrefix("/ui", http.FileServer(http.FS(webFS)))
	protected.GET("/ui/*filepath", gin.WrapH(uiFileServer))
	protected.HEAD("/ui/*filepath", gin.WrapH(uiFileServer))
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
		protected.GET("/api/v1/search", recallHandler.Search)
	}

	if inspectHandler != nil {
		protected.GET("/api/v1/inspect/cat", inspectHandler.Cat)
		protected.GET("/api/v1/inspect/tree", inspectHandler.Tree)
		protected.GET("/api/v1/inspect/meta", inspectHandler.Meta)
	}

	if assertionHandler != nil {
		protected.GET("/api/v1/assertions", assertionHandler.List)
		protected.POST("/api/v1/assertions", assertionHandler.Create)
		protected.DELETE("/api/v1/assertions", assertionHandler.Retract)
	}

	if backfillHandler != nil {
		protected.POST("/api/v1/backfill/t1", backfillHandler.T1)
		protected.POST("/api/v1/backfill/t2", backfillHandler.T2)
		protected.POST("/api/v1/backfill/t3", backfillHandler.T3)
	}

	if embedHandler != nil {
		protected.GET("/api/v1/embed/status", embedHandler.Status)
	}

	return r, nil
}
