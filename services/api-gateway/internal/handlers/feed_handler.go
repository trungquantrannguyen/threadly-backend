package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/response"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/client"
)

type FeedHandler struct {
	FeedClient *client.FeedClient
	log        zerolog.Logger
}

func NewFeedHandler(FeedClient *client.FeedClient, log zerolog.Logger) *FeedHandler {
	return &FeedHandler{
		FeedClient: FeedClient,
		log:        log,
	}
}

func (h *FeedHandler) GetHealth(c *gin.Context) {
	health, err := h.FeedClient.GetHealth(c.Request.Context())
	if err != nil {
		h.log.Error().
			Err(err).
			Msg("Failed to call feed service health grpc method")

		response.Error(c, http.StatusServiceUnavailable, "Feed service unavailable", err)
		return
	}

	response.OK(c, http.StatusOK, "Feed service available", health)
}
