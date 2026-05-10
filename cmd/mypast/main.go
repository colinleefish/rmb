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
	"github.com/colinleefish/mypast/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	runCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	runner := cli.Runner{
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

			if err := db.AutoMigrate(database); err != nil {
				return fmt.Errorf("db migrate: %w", err)
			}

			healthSvc := service.NewHealthService(database)
			sessionUploadSvc := service.NewSessionUploadService(database)

			if cfg.Summarizer.Enabled {
				llmClient, err := llm.NewOpenAICompatibleClient(llm.OpenAICompatibleConfig{
					Provider:   cfg.VLM.Provider,
					APIBase:    cfg.VLM.APIBase,
					APIKey:     cfg.VLM.APIKey,
					Model:      cfg.VLM.Model,
					MaxRetries: cfg.VLM.MaxRetries,
					Timeout:    cfg.VLM.Timeout,
				})
				if err != nil {
					return fmt.Errorf("init llm client: %w", err)
				}

				worker := service.NewSummarizationWorker(database, llmClient, cfg.Summarizer)
				go func() {
					if err := worker.Run(ctx); err != nil {
						log.Printf("summarization worker exited with error: %v", err)
					}
				}()
			}

			healthHandler := handler.NewHealthHandler(healthSvc)
			sessionUploadHandler := handler.NewSessionUploadHandler(sessionUploadSvc)

			httpRouter := router.New(healthHandler, sessionUploadHandler)

			return server.RunHTTP(ctx, cfg.Server, httpRouter)
		},
	}

	if err := runner.Run(runCtx, os.Args[1:]); err != nil {
		log.Fatalf("run command: %v", err)
	}
}
