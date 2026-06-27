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

// GetHealth godoc
// @Summary Get Feed service health
// @Description Get the status of feed service
// @Tags Feeds
// @Accept json
// @Produce json
// @Success 200 {object} dto.GetFeedServiceHealthResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /feeds/health [get]
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
