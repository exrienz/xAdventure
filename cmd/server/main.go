package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/muz/xadventure/internal/config"
	"github.com/muz/xadventure/internal/llm"
	"github.com/muz/xadventure/internal/repository"
	"github.com/muz/xadventure/internal/service"
	"github.com/muz/xadventure/internal/transport"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting Infinite Narrative Engine",
		"version", "2.7.0",
		"addr", "0.0.0.0:"+cfg.Port,
		"temperature", cfg.LLMTemperature,
		"top_p", cfg.LLMTopP,
	)

	db, err := sql.Open("sqlite", cfg.DBPath+"?_pragma=journal_mode(WAL)")
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := runMigrations(db); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	repo, err := repository.NewSQLiteRepo(cfg.DBPath)
	if err != nil {
		slog.Error("failed to create repository", "error", err)
		os.Exit(1)
	}
	defer repo.Close()

	llmClient := llm.NewClient(cfg.OpenAIBase, cfg.OpenAIKey, cfg.OpenAIModel, cfg.LLMTimeoutSec, cfg.LLMMaxRetries, cfg.LLMTemperature, cfg.LLMTopP)
	engine := service.NewEngine(repo, llmClient, cfg)

	router := transport.SetupRouter(engine, cfg)

	srv := &http.Server{
		Addr:    "0.0.0.0:" + cfg.Port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
}

func runMigrations(db *sql.DB) error {
	schema1, err := os.ReadFile("./migrations/001_init.up.sql")
	if err != nil {
		return err
	}
	if _, err = db.Exec(string(schema1)); err != nil {
		return err
	}

	// Basic check to see if column exists before trying to add it
	var columnName string
	err = db.QueryRow("SELECT name FROM pragma_table_info('story_logs') WHERE name='color_coded_content'").Scan(&columnName)
	if err == sql.ErrNoRows {
		schema2, err := os.ReadFile("./migrations/002_add_color_coded_content.up.sql")
		if err != nil {
			return err
		}
		if _, err = db.Exec(string(schema2)); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}
