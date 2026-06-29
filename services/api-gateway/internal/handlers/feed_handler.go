package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/middleware"
	"github.com/trungquantrannguyen/threadly/pkg/response"
	feedpb "github.com/trungquantrannguyen/threadly/proto/feed"
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

// GetHomeFeed godoc
// @Summary Get home feed
// @Description Returns the authenticated user's home feed.
// @Tags Feeds
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit"
// @Param cursor query string false "Cursor"
// @Success 200 {object} dto.HomeFeedResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /feeds/home [get]
func (h *FeedHandler) GetHomeFeed(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	limit := int32(20)
	if value := c.Query("limit"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil || parsed <= 0 || parsed > 50 {
			response.Error(c, http.StatusBadRequest, "Invalid limit", err)
			return
		}
		limit = int32(parsed)
	}

	res, err := h.FeedClient.GetHomeFeed(c.Request.Context(), &feedpb.GetHomeFeedRequest{
		UserId: userID,
		Limit:  limit,
		Cursor: c.Query("cursor"),
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get home feed")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusOK, "Get home feed successfully", res)
}
