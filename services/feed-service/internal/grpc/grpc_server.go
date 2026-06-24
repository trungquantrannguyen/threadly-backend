package grpc

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	feedpb "github.com/trungquantrannguyen/threadly/proto/feed"
)

type FeedServiceServer struct {
	feedpb.UnimplementedFeedServiceServer
	cfg config.Config
	log zerolog.Logger
}

func NewContentServiceServer(cfg config.Config, log zerolog.Logger) *FeedServiceServer {
	return &FeedServiceServer{
		cfg: cfg,
		log: log,
	}
}

func (s *FeedServiceServer) GetHealth(ctx context.Context, req *feedpb.GetFeedServiceHealthRequest) (*feedpb.GetFeedServiceHealthResponse, error) {
	s.log.Info().Msg("Feed service is healthy")
	return &feedpb.GetFeedServiceHealthResponse{
		Status:    "ok",
		Service:   s.cfg.ServiceName,
		Env:       s.cfg.AppEnv,
		CheckedAt: time.Now().String(),
	}, nil
}
