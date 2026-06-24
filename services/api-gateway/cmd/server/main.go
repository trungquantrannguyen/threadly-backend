package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/trungquantrannguyen/threadly/pkg/config"
	"github.com/trungquantrannguyen/threadly/pkg/logger"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/app"
)

func main() {
	cfg := config.Load("api-gateway", "8080")
	log := logger.New(cfg.ServiceName, cfg.AppEnv)

	router, cleanup := app.NewRouter(cfg, log)

	defer func() {
		if err := cleanup(); err != nil {
			log.Error().Err(err).Msg("Failed to cleanup api gateway resources")
		}
	}()

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Info().
		Str("port", cfg.Port).
		Str("env", cfg.AppEnv).
		Msg("Starting api gateway")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().
			Err(err).
			Msg("Failed to start api gateway")
	}
}
