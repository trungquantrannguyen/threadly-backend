package handlers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/middleware"
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

// GetHealth godoc
// @Summary Get Storage service health
// @Description Get the status of storage service
// @Tags Storage
// @Accept json
// @Produce json
// @Success 200 {object} dto.GetStorageServiceHealthResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /storages/health [get]
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

// UploadMedia godoc
// @Summary Upload media
// @Description Upload an image file
// @Tags Storage
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Media file"
// @Success 201 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 503 {object} response.ErrorResponse
// @Router /storages/upload [post]
func (h *StorageHandler) UploadMedia(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "file is required", err)
		return
	}

	const maxFileSize = 5 << 20 // 5MB
	if fileHeader.Size > maxFileSize {
		response.Error(c, http.StatusBadRequest, "file size must be less than 5MB", nil)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to open file", err)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to read file", err)
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")

	res, err := h.storageClient.UploadMedia(
		c.Request.Context(),
		userID,
		fileHeader.Filename,
		contentType,
		content,
	)
	if err != nil {
		h.log.Error().Err(err).Msg("failed to upload media")
		response.Error(c, http.StatusServiceUnavailable, "Storage service unavailable", err)
		return
	}

	response.OK(c, http.StatusCreated, "Media uploaded successfully", res)
}
