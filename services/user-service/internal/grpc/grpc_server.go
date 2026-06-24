package grpc

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	userpb "github.com/trungquantrannguyen/threadly/proto/user"
)

type UserServiceServer struct {
	userpb.UnimplementedUserServiceServer
	cfg config.Config
	log zerolog.Logger
}

func NewUserServiceServer(cfg config.Config, log zerolog.Logger) *UserServiceServer {
	return &UserServiceServer{
		cfg: cfg,
		log: log,
	}
}

func (s *UserServiceServer) GetHealth(ctx context.Context, req *userpb.GetUserServiceHealthRequest) (*userpb.GetUserServiceHealthResponse, error) {
	s.log.Info().Msg("User service is healthy")
	return &userpb.GetUserServiceHealthResponse{
		Status:    "ok",
		Service:   s.cfg.ServiceName,
		Env:       s.cfg.AppEnv,
		CheckedAt: time.Now().String(),
	}, nil
}
