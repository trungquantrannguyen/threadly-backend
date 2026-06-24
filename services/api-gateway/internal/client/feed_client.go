package client

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	feedpb "github.com/trungquantrannguyen/threadly/proto/feed"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type FeedClient struct {
	conn   *grpc.ClientConn
	client feedpb.FeedServiceClient
	log    zerolog.Logger
}

func NewFeedClient(cfg config.Config, log zerolog.Logger) (*FeedClient, error) {
	conn, err := grpc.NewClient(
		cfg.FeedServiceGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	client := feedpb.NewFeedServiceClient(conn)

	return &FeedClient{
		conn:   conn,
		client: client,
		log:    log,
	}, nil
}

func (c *FeedClient) GetHealth(ctx context.Context) (*feedpb.GetFeedServiceHealthResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.GetHealth(ctx, &feedpb.GetFeedServiceHealthRequest{})
}

func (c *FeedClient) Close() error {
	return c.conn.Close()
}
