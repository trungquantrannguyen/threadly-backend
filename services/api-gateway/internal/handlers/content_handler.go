package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/response"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/client"
)

type ContentHandler struct {
	contentClient *client.ContentClient
	log           zerolog.Logger
}

func NewContentHandler(contentClient *client.ContentClient, log zerolog.Logger) *ContentHandler {
	return &ContentHandler{
		contentClient: contentClient,
		log:           log,
	}
}

func (h *ContentHandler) GetHealth(c *gin.Context) {
	health, err := h.contentClient.GetHealth(c.Request.Context())
	if err != nil {
		h.log.Error().
			Err(err).
			Msg("Failed to call content service health grpc method")

		response.Error(c, http.StatusServiceUnavailable, "Content service unavailable", err)
		return
	}

	response.OK(c, http.StatusOK, "Content service available", health)
}
