package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"github.com/trungquantrannguyen/threadly/services/storage-service/internal/provider"
	"github.com/trungquantrannguyen/threadly/services/storage-service/internal/repository"
)

var (
	ErrInvalidUploaderID = errors.New("invalid uploader id")
	ErrEmptyFile         = errors.New("file is empty")
	ErrInvalidMimeType   = errors.New("invalid file type")
)

type UploadMediaRequest struct {
	UploaderID  string
	Filename    string
	ContentType string
	Content     []byte
}

type UploadMediaResponse struct {
	ID         string
	URL        string
	StorageKey string
	MimeType   string
	SizeBytes  int64
	CreatedAt  string
}

type StorageService interface {
	UploadMedia(ctx context.Context, req UploadMediaRequest) (*UploadMediaResponse, error)
}

type storageService struct {
	mediaRepo       repository.MediaRepository
	storageProvider provider.SupabaseStorageProvider
}

func NewStorageService(
	mediaRepo repository.MediaRepository,
	storageProvider provider.SupabaseStorageProvider,
) StorageService {
	return &storageService{
		mediaRepo:       mediaRepo,
		storageProvider: storageProvider,
	}
}

func (s *storageService) UploadMedia(ctx context.Context, req UploadMediaRequest) (*UploadMediaResponse, error) {
	uploaderID, err := uuid.Parse(req.UploaderID)
	if err != nil {
		return nil, ErrInvalidUploaderID
	}

	if len(req.Content) == 0 {
		return nil, ErrEmptyFile
	}

	if !isAllowedImageType(req.ContentType) {
		return nil, ErrInvalidMimeType
	}

	ext := strings.ToLower(filepath.Ext(req.Filename))
	if ext == "" {
		ext = extensionFromMime(req.ContentType)
	}

	storageKey := fmt.Sprintf("users/%s/uploads/%s%s", uploaderID.String(), uuid.NewString(), ext)

	url, err := s.storageProvider.Upload(ctx, storageKey, req.ContentType, req.Content)
	if err != nil {
		return nil, err
	}

	media := &dbmodel.Media{
		UploaderID: uploaderID,
		URL:        url,
		StorageKey: storageKey,
		MimeType:   req.ContentType,
		SizeBytes:  int64(len(req.Content)),
		Status:     "uploaded",
		CreatedAt:  time.Now().UTC(),
	}

	if err := s.mediaRepo.Create(ctx, media); err != nil {
		return nil, err
	}

	return &UploadMediaResponse{
		ID:         media.ID.String(),
		URL:        media.URL,
		StorageKey: media.StorageKey,
		MimeType:   media.MimeType,
		SizeBytes:  media.SizeBytes,
		CreatedAt:  media.CreatedAt.Format(time.RFC3339),
	}, nil
}

func isAllowedImageType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

func extensionFromMime(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".bin"
	}
}
