package client

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	contentpb "github.com/trungquantrannguyen/threadly/proto/content"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ContentClient struct {
	conn   *grpc.ClientConn
	client contentpb.ContentServiceClient
	log    zerolog.Logger
}

func NewContentClient(cfg config.Config, log zerolog.Logger) (*ContentClient, error) {
	conn, err := grpc.NewClient(
		cfg.ContentServiceGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	client := contentpb.NewContentServiceClient(conn)

	return &ContentClient{
		conn:   conn,
		client: client,
		log:    log,
	}, nil
}

func (c *ContentClient) GetHealth(ctx context.Context) (*contentpb.GetContentServiceHealthResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.GetHealth(ctx, &contentpb.GetContentServiceHealthRequest{})
}

func (c *ContentClient) Close() error {
	return c.conn.Close()
}
