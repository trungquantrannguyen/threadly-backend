package grpc

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	storagepb "github.com/trungquantrannguyen/threadly/proto/storage"
)

type StorageServiceServer struct {
	storagepb.UnimplementedStorageServiceServer
	cfg config.Config
	log zerolog.Logger
}

func NewStorageServiceServer(cfg config.Config, log zerolog.Logger) *StorageServiceServer {
	return &StorageServiceServer{
		cfg: cfg,
		log: log,
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
