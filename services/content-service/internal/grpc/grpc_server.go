package grpc

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	contentpb "github.com/trungquantrannguyen/threadly/proto/content"
)

type ContentServiceServer struct {
	contentpb.UnimplementedContentServiceServer
	cfg config.Config
	log zerolog.Logger
}

func NewContentServiceServer(cfg config.Config, log zerolog.Logger) *ContentServiceServer {
	return &ContentServiceServer{
		cfg: cfg,
		log: log,
	}
}

func (s *ContentServiceServer) GetHealth(ctx context.Context, req *contentpb.GetContentServiceHealthRequest) (*contentpb.GetContentServiceHealthResponse, error) {
	s.log.Info().Msg("Content service is healthy")
	return &contentpb.GetContentServiceHealthResponse{
		Status:    "ok",
		Service:   s.cfg.ServiceName,
		Env:       s.cfg.AppEnv,
		CheckedAt: time.Now().String(),
	}, nil
}
