package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/colinleefish/mypast/internal/cli"
	"github.com/colinleefish/mypast/internal/config"
	"github.com/colinleefish/mypast/internal/db"
	"github.com/colinleefish/mypast/internal/http/handler"
	"github.com/colinleefish/mypast/internal/http/router"
	"github.com/colinleefish/mypast/internal/llm"
	"github.com/colinleefish/mypast/internal/server"
	"github.com/colinleefish/mypast/internal/service/browse"
	"github.com/colinleefish/mypast/internal/service/health"
	"github.com/colinleefish/mypast/internal/service/extract"
	"github.com/colinleefish/mypast/internal/service/session"
	"github.com/colinleefish/mypast/internal/service/summarize"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	runCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	runner := cli.Runner{
		Config: cfg,
		Serve: func(ctx context.Context) error {
			database, err := db.New(ctx, cfg.DB.URL)
			if err != nil {
				return fmt.Errorf("db connect: %w", err)
			}
			sqlDB, err := database.DB()
			if err != nil {
				return fmt.Errorf("get db handle: %w", err)
			}
			defer sqlDB.Close()

			if err := db.Migrate(ctx, database); err != nil {
				return fmt.Errorf("db migrate: %w", err)
			}

			healthSvc := health.NewService(database)
			sessionUploadSvc := session.NewUploadService(database)

			if cfg.Extraction.Enabled || cfg.Summarizer.Enabled {
				llmClient, err := llm.NewOpenAICompatibleClient(llm.OpenAICompatibleConfig{
					Provider:   cfg.LLM.Provider,
					APIBase:    cfg.LLM.APIBase,
					APIKey:     cfg.LLM.APIKey,
					Model:      cfg.LLM.Model,
					MaxRetries: cfg.LLM.MaxRetries,
					Timeout:    cfg.LLM.Timeout,
				})
				if err != nil {
					return fmt.Errorf("init llm client: %w", err)
				}

				if cfg.Extraction.Enabled {
					t1Worker := extract.NewWorker(database, llmClient, cfg.Extraction)
					go func() {
						if err := t1Worker.Run(ctx); err != nil {
							log.Printf("t1 extraction worker exited with error: %v", err)
						}
					}()
				}

				if cfg.Summarizer.Enabled {
					worker := summarize.NewWorker(database, llmClient, cfg.Summarizer)
					go func() {
						if err := worker.Run(ctx); err != nil {
							log.Printf("summarization worker exited with error: %v", err)
						}
					}()
				}
			}

			browseSvc := browse.NewService(database)
			healthHandler := handler.NewHealthHandler(healthSvc)
			sessionUploadHandler := handler.NewSessionUploadHandler(sessionUploadSvc)
			browseHandler := handler.NewBrowseHandler(browseSvc)

			httpRouter, err := router.New(cfg, healthHandler, sessionUploadHandler, browseHandler)
			if err != nil {
				return fmt.Errorf("build router: %w", err)
			}

			return server.RunHTTP(ctx, cfg.Server, httpRouter)
		},
	}

	if err := runner.Run(runCtx, os.Args[1:]); err != nil {
		log.Fatalf("run command: %v", err)
	}
}
