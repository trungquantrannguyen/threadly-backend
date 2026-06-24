package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	"github.com/trungquantrannguyen/threadly/pkg/response"
)

type HealthHandler struct {
	cfg config.Config
}

func NewHealthHandler(cfg config.Config) *HealthHandler {
	return &HealthHandler{
		cfg: cfg,
	}
}

func (h *HealthHandler) Check(ctx *gin.Context) {
	response.OK(ctx, http.StatusOK, "service is health", gin.H{
		"service":   h.cfg.ServiceName,
		"env":       h.cfg.AppEnv,
		"timestamp": time.Now().UTC(),
	})
}
