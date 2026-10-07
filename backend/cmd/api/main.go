package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"backend_voxtext/internal/config"
	"backend_voxtext/internal/modules/auth"
	"backend_voxtext/internal/modules/middleware"
	"backend_voxtext/internal/modules/voxtext"
	"backend_voxtext/internal/pkg/accesstoken"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseAddress)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	voxtextConfig, err := voxtext.LoadConfig()
	if err != nil {
		return fmt.Errorf("load transcription configuration: %w", err)
	}
	voxtextModule, err := voxtext.NewModule(voxtextConfig)
	if err != nil {
		return fmt.Errorf("initialize transcription module: %w", err)
	}
	defer voxtextModule.Close()

	tokens, err := accesstoken.NewManager([]byte(cfg.JWTSecret), 24*time.Hour)
	if err != nil {
		return fmt.Errorf("configure access tokens: %w", err)
	}
	repository := auth.NewRepository(pool)
	service := auth.NewService(repository, tokens)
	handler := auth.NewHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/register", handler.Register)
	mux.HandleFunc("POST /auth/login", handler.Login)
	mux.Handle("GET /auth/me", middleware.RequireAuth(tokens)(http.HandlerFunc(handler.Me)))
	mux.Handle("POST /v1/transcriptions", middleware.RequireAuth(tokens)(http.HandlerFunc(voxtextModule.Handler.Transcribe)))
	frontendDir, err := filepath.Abs(filepath.Join("..", "frontend"))
	if err != nil {
		return fmt.Errorf("resolve frontend directory: %w", err)
	}
	indexInfo, err := os.Stat(filepath.Join(frontendDir, "index.html"))
	if err != nil {
		return fmt.Errorf("frontend index.html is unavailable in %q: %w", frontendDir, err)
	}
	if !indexInfo.Mode().IsRegular() {
		return fmt.Errorf("frontend index.html is not a regular file in %q", frontendDir)
	}
	mux.Handle("GET /", http.FileServer(http.Dir(frontendDir)))

	server := &http.Server{
		Addr:              cfg.ServerAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	serveError := make(chan error, 1)
	go func() {
		serveError <- server.ListenAndServe()
	}()
	log.Printf("HTTP server listening on %s", cfg.ServerAddress)

	select {
	case err := <-serveError:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}
