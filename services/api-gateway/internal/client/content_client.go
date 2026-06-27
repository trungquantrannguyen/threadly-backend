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

func (c *ContentClient) CreatePost(ctx context.Context, req *contentpb.CreatePostRequest) (*contentpb.PostResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.CreatePost(ctx, req)
}

func (c *ContentClient) GetPost(ctx context.Context, req *contentpb.GetPostRequest) (*contentpb.PostResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.GetPost(ctx, req)
}

func (c *ContentClient) DeletePost(ctx context.Context, req *contentpb.DeletePostRequest) (*contentpb.DeletePostResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.DeletePost(ctx, req)
}

func (c *ContentClient) CreateReply(ctx context.Context, req *contentpb.CreateReplyRequest) (*contentpb.PostResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.CreateReply(ctx, req)
}

func (c *ContentClient) GetReplies(ctx context.Context, req *contentpb.GetRepliesRequest) (*contentpb.PostListResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.GetReplies(ctx, req)
}

func (c *ContentClient) Close() error {
	return c.conn.Close()
}
