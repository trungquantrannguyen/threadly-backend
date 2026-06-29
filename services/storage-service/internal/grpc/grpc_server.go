package grpc

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	storagepb "github.com/trungquantrannguyen/threadly/proto/storage"
	"github.com/trungquantrannguyen/threadly/services/storage-service/internal/service"
)

type StorageServiceServer struct {
	storagepb.UnimplementedStorageServiceServer
	cfg            config.Config
	log            zerolog.Logger
	storageService service.StorageService
}

func NewStorageServiceServer(cfg config.Config, log zerolog.Logger, storageService service.StorageService) *StorageServiceServer {
	return &StorageServiceServer{
		cfg:            cfg,
		log:            log,
		storageService: storageService,
	}
}

func (s *StorageServiceServer) GetHealth(ctx context.Context, req *storagepb.GetStorageServiceHealthRequest) (*storagepb.GetStorageServiceHealthResponse, error) {
	s.log.Info().Msg("storage service is healthy")
	return &storagepb.GetStorageServiceHealthResponse{
		Status:    "ok",
		Service:   s.cfg.ServiceName,
		Env:       s.cfg.AppEnv,
		CheckedAt: time.Now().String(),
	}, nil
}

func (s *StorageServiceServer) UploadMedia(ctx context.Context, req *storagepb.UploadMediaRequest) (*storagepb.UploadMediaResponse, error) {
	media, err := s.storageService.UploadMedia(ctx, service.UploadMediaRequest{
		UploaderID:  req.GetUploaderId(),
		Filename:    req.GetFilename(),
		ContentType: req.GetContentType(),
		Content:     req.GetContent(),
	})
	if err != nil {
		return nil, err
	}

	return &storagepb.UploadMediaResponse{
		Id:         media.ID,
		Url:        media.URL,
		StorageKey: media.StorageKey,
		MimeType:   media.MimeType,
		SizeBytes:  media.SizeBytes,
		CreatedAt:  media.CreatedAt,
	}, nil
}
