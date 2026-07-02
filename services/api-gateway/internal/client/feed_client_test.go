package client

import (
	"context"
	"net"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	feedpb "github.com/trungquantrannguyen/threadly/proto/feed"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type fakeFeedServiceServer struct {
	feedpb.UnimplementedFeedServiceServer

	healthCalled      bool
	getHomeFeedCalled bool
}

func startFakeFeedGRPCServer(t *testing.T, fakeServer *fakeFeedServiceServer) (*FeedClient, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	feedpb.RegisterFeedServiceServer(grpcServer, fakeServer)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	client := &FeedClient{
		conn:   conn,
		client: feedpb.NewFeedServiceClient(conn),
		log:    zerolog.Nop(),
	}

	cleanup := func() {
		_ = client.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}

	return client, cleanup
}

func (s *fakeFeedServiceServer) GetHealth(ctx context.Context, req *feedpb.GetFeedServiceHealthRequest) (*feedpb.GetFeedServiceHealthResponse, error) {
	s.healthCalled = true

	return &feedpb.GetFeedServiceHealthResponse{
		Status:    "ok",
		Service:   "feed-service",
		Env:       "test",
		CheckedAt: "now",
	}, nil
}

func (s *fakeFeedServiceServer) GetHomeFeed(ctx context.Context, req *feedpb.GetHomeFeedRequest) (*feedpb.HomeFeedResponse, error) {
	s.getHomeFeedCalled = true

	return &feedpb.HomeFeedResponse{
		Posts: []*feedpb.FeedPostResponse{
			{
				Id:            "post-1",
				AuthorId:      "author-1",
				Content:       "hello feed",
				Visibility:    "public",
				LikeCount:     1,
				ReplyCount:    2,
				RepostCount:   3,
				BookmarkCount: 4,
				CreatedAt:     "2026-07-02T10:00:00Z",
				UpdatedAt:     "2026-07-02T10:01:00Z",
				Author: &feedpb.FeedUserSummary{
					Id:          "author-1",
					Username:    "author",
					DisplayName: "Author User",
					AvatarUrl:   "https://example.com/avatar.png",
					IsVerified:  true,
				},
				Media: []*feedpb.FeedMediaResponse{
					{
						Id:        "media-1",
						Url:       "https://example.com/media.png",
						MimeType:  "image/png",
						SizeBytes: 1024,
						Width:     640,
						Height:    480,
					},
				},
			},
		},
		NextCursor: "next-cursor",
	}, nil
}

func TestFeedClientGetHealth(t *testing.T) {
	fakeServer := &fakeFeedServiceServer{}
	client, cleanup := startFakeFeedGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.GetHealth(context.Background())

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.healthCalled)
	assert.Equal(t, "ok", res.Status)
	assert.Equal(t, "feed-service", res.Service)
	assert.Equal(t, "test", res.Env)
}

func TestFeedClientGetHomeFeed(t *testing.T) {
	fakeServer := &fakeFeedServiceServer{}
	client, cleanup := startFakeFeedGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.GetHomeFeed(context.Background(), &feedpb.GetHomeFeedRequest{
		UserId: "user-1",
		Limit:  20,
		Cursor: "cursor-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.getHomeFeedCalled)
	assert.Equal(t, "next-cursor", res.NextCursor)

	require.Len(t, res.Posts, 1)
	assert.Equal(t, "post-1", res.Posts[0].Id)
	assert.Equal(t, "hello feed", res.Posts[0].Content)
	require.NotNil(t, res.Posts[0].Author)
	assert.Equal(t, "author", res.Posts[0].Author.Username)
	require.Len(t, res.Posts[0].Media, 1)
	assert.Equal(t, "media-1", res.Posts[0].Media[0].Id)
}
