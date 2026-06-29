package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/middleware"
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

// GetHealth godoc
// @Summary Get Notification service health
// @Description Get the status of notification service
// @Tags Notifications
// @Accept json
// @Produce json
// @Success 200 {object} dto.GetNotificationServiceHealthResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /notifications/health [get]
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

// GetNotifications godoc
// @Summary Get my notifications
// @Description Get notifications for the authenticated user
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Limit"
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /notifications [get]
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	limit := int32(20)
	if value := c.Query("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}

	res, err := h.notificationClient.GetNotifications(c.Request.Context(), userID, limit)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get notifications")
		response.Error(c, http.StatusServiceUnavailable, "Notification service unavailable", err)
		return
	}

	response.OK(c, http.StatusOK, "Notifications fetched successfully", res)
}

// MarkNotificationRead godoc
// @Summary Mark notification as read
// @Description Mark one notification as read for the authenticated user
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /notifications/{id}/read [patch]
func (h *NotificationHandler) MarkNotificationRead(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	notificationID := c.Param("id")

	res, err := h.notificationClient.MarkNotificationRead(c.Request.Context(), userID, notificationID)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to mark notification as read")
		response.Error(c, http.StatusServiceUnavailable, "Notification service unavailable", err)
		return
	}

	response.OK(c, http.StatusOK, res.GetMessage(), res)
}

// MarkAllNotificationsRead godoc
// @Summary Mark all notifications as read
// @Description Mark all unread notifications as read for the authenticated user
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /notifications/read-all [patch]
func (h *NotificationHandler) MarkAllNotificationsRead(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	res, err := h.notificationClient.MarkAllNotificationsRead(c.Request.Context(), userID)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to mark all notifications as read")
		response.Error(c, http.StatusServiceUnavailable, "Notification service unavailable", err)
		return
	}

	response.OK(c, http.StatusOK, res.GetMessage(), res)
}
