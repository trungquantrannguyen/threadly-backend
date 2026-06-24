package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/response"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/client"
)

type NotificationHandler struct {
	notificationClient *client.NotificationClient
	log                zerolog.Logger
}

func NewNotificationHandler(notificationClient *client.NotificationClient, log zerolog.Logger) *NotificationHandler {
	return &NotificationHandler{
		notificationClient: notificationClient,
		log:                log,
	}
}

func (h *NotificationHandler) GetHealth(c *gin.Context) {
	health, err := h.notificationClient.GetHealth(c.Request.Context())
	if err != nil {
		h.log.Error().
			Err(err).
			Msg("Failed to call Notification service health grpc method")

		response.Error(c, http.StatusServiceUnavailable, "Notification service unavailable", err)
		return
	}

	response.OK(c, http.StatusOK, "Notification service available", health)
}
