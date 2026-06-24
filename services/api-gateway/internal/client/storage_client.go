package client

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	storagepb "github.com/trungquantrannguyen/threadly/proto/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type StorageClient struct {
	conn   *grpc.ClientConn
	client storagepb.StorageServiceClient
	log    zerolog.Logger
}

func NewStorageClient(cfg config.Config, log zerolog.Logger) (*StorageClient, error) {
	conn, err := grpc.NewClient(
		cfg.StorageServiceGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	client := storagepb.NewStorageServiceClient(conn)

	return &StorageClient{
		conn:   conn,
		client: client,
		log:    log,
	}, nil
}

func (c *StorageClient) GetHealth(ctx context.Context) (*storagepb.GetStorageServiceHealthResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.GetHealth(ctx, &storagepb.GetStorageServiceHealthRequest{})
}

func (c *StorageClient) Close() error {
	return c.conn.Close()
}
