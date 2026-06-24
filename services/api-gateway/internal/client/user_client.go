package client

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	userpb "github.com/trungquantrannguyen/threadly/proto/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserClient struct {
	conn   *grpc.ClientConn
	client userpb.UserServiceClient
	log    zerolog.Logger
}

func NewUserClient(cfg config.Config, log zerolog.Logger) (*UserClient, error) {
	conn, err := grpc.NewClient(
		cfg.UserServiceGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	client := userpb.NewUserServiceClient(conn)

	return &UserClient{
		conn:   conn,
		client: client,
		log:    log,
	}, nil
}

func (c *UserClient) GetHealth(ctx context.Context) (*userpb.GetUserServiceHealthResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.GetHealth(ctx, &userpb.GetUserServiceHealthRequest{})
}

func (c *UserClient) Close() error {
	return c.conn.Close()
}
