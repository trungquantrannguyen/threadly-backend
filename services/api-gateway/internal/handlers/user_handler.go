package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/response"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/client"
)

type UserHandler struct {
	userClient *client.UserClient
	log        zerolog.Logger
}

func NewUserHandler(userClient *client.UserClient, log zerolog.Logger) *UserHandler {
	return &UserHandler{
		userClient: userClient,
		log:        log,
	}
}

func (h *UserHandler) GetHealth(c *gin.Context) {
	health, err := h.userClient.GetHealth(c.Request.Context())
	if err != nil {
		h.log.Error().
			Err(err).
			Msg("failed to call user service health grpc method")

		response.Error(c, http.StatusServiceUnavailable, "user service unavailable", err)
		return
	}

	response.OK(c, http.StatusOK, "user service available", health)
}
