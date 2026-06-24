package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/response"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/client"
)

type StorageHandler struct {
	storageClient *client.StorageClient
	log           zerolog.Logger
}

func NewStorageHandler(storageClient *client.StorageClient, log zerolog.Logger) *StorageHandler {
	return &StorageHandler{
		storageClient: storageClient,
		log:           log,
	}
}

func (h *StorageHandler) GetHealth(c *gin.Context) {
	health, err := h.storageClient.GetHealth(c.Request.Context())
	if err != nil {
		h.log.Error().
			Err(err).
			Msg("failed to call Storage service health grpc method")

		response.Error(c, http.StatusServiceUnavailable, "Storage service unavailable", err)
		return
	}

	response.OK(c, http.StatusOK, "Storage service available", health)
}
