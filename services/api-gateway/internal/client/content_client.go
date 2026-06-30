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

func (c *ContentClient) UpdatePost(ctx context.Context, req *contentpb.UpdatePostRequest) (*contentpb.PostResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.UpdatePost(ctx, req)
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

func (c *ContentClient) LikePost(ctx context.Context, req *contentpb.LikePostRequest) (*contentpb.ActionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.LikePost(ctx, req)
}

func (c *ContentClient) UnlikePost(ctx context.Context, req *contentpb.UnlikePostRequest) (*contentpb.ActionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.UnlikePost(ctx, req)
}

func (c *ContentClient) BookmarkPost(ctx context.Context, req *contentpb.BookmarkPostRequest) (*contentpb.ActionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.BookmarkPost(ctx, req)
}

func (c *ContentClient) UnbookmarkPost(ctx context.Context, req *contentpb.UnbookmarkPostRequest) (*contentpb.ActionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.UnbookmarkPost(ctx, req)
}

func (c *ContentClient) RepostPost(ctx context.Context, req *contentpb.RepostPostRequest) (*contentpb.ActionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.RepostPost(ctx, req)
}

func (c *ContentClient) UndoRepost(ctx context.Context, req *contentpb.UndoRepostRequest) (*contentpb.ActionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.UndoRepost(ctx, req)
}

func (c *ContentClient) FollowUser(ctx context.Context, req *contentpb.FollowUserRequest) (*contentpb.ActionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.FollowUser(ctx, req)
}

func (c *ContentClient) UnfollowUser(ctx context.Context, req *contentpb.UnfollowUserRequest) (*contentpb.ActionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.UnfollowUser(ctx, req)
}

func (c *ContentClient) GetFollowers(ctx context.Context, req *contentpb.GetFollowersRequest) (*contentpb.UserListResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.GetFollowers(ctx, req)
}

func (c *ContentClient) GetFollowing(ctx context.Context, req *contentpb.GetFollowingRequest) (*contentpb.UserListResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.GetFollowing(ctx, req)
}

func (c *ContentClient) GetUserTimeline(ctx context.Context, req *contentpb.GetUserTimelineRequest) (*contentpb.TimelineResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.GetUserTimeline(ctx, req)
}

func (c *ContentClient) Close() error {
	return c.conn.Close()
}
