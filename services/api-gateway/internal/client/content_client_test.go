package client

import (
	"context"
	"net"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	contentpb "github.com/trungquantrannguyen/threadly/proto/content"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type fakeContentServiceServer struct {
	contentpb.UnimplementedContentServiceServer

	healthCalled          bool
	createPostCalled      bool
	getPostCalled         bool
	updatePostCalled      bool
	deletePostCalled      bool
	createReplyCalled     bool
	getRepliesCalled      bool
	likePostCalled        bool
	unlikePostCalled      bool
	bookmarkPostCalled    bool
	unbookmarkPostCalled  bool
	repostPostCalled      bool
	undoRepostCalled      bool
	followUserCalled      bool
	unfollowUserCalled    bool
	getFollowersCalled    bool
	getFollowingCalled    bool
	getUserTimelineCalled bool
}

func startFakeContentGRPCServer(t *testing.T, fakeServer *fakeContentServiceServer) (*ContentClient, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	contentpb.RegisterContentServiceServer(grpcServer, fakeServer)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	client := &ContentClient{
		conn:   conn,
		client: contentpb.NewContentServiceClient(conn),
		log:    zerolog.Nop(),
	}

	cleanup := func() {
		_ = client.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}

	return client, cleanup
}

func fakePostResponse() *contentpb.PostResponse {
	return &contentpb.PostResponse{
		Id:            "post-1",
		AuthorId:      "author-1",
		ReplyToPostId: "",
		Content:       "hello",
		Visibility:    "public",
		LikeCount:     1,
		ReplyCount:    2,
		RepostCount:   3,
		BookmarkCount: 4,
		CreatedAt:     "2026-07-02T10:00:00Z",
		UpdatedAt:     "2026-07-02T10:01:00Z",
		Author: &contentpb.UserSummary{
			Id:          "author-1",
			Username:    "author",
			DisplayName: "Author User",
			AvatarUrl:   "https://example.com/avatar.png",
			IsVerified:  true,
		},
		Media: []*contentpb.MediaResponse{
			{
				Id:        "media-1",
				Url:       "https://example.com/media.png",
				MimeType:  "image/png",
				SizeBytes: 1024,
				Width:     640,
				Height:    480,
			},
		},
	}
}

func fakeActionResponse(message string) *contentpb.ActionResponse {
	return &contentpb.ActionResponse{
		Success: true,
		Message: message,
	}
}

func (s *fakeContentServiceServer) GetHealth(ctx context.Context, req *contentpb.GetContentServiceHealthRequest) (*contentpb.GetContentServiceHealthResponse, error) {
	s.healthCalled = true

	return &contentpb.GetContentServiceHealthResponse{
		Status:    "ok",
		Service:   "content-service",
		Env:       "test",
		CheckedAt: "now",
	}, nil
}

func (s *fakeContentServiceServer) CreatePost(ctx context.Context, req *contentpb.CreatePostRequest) (*contentpb.PostResponse, error) {
	s.createPostCalled = true

	post := fakePostResponse()
	post.AuthorId = req.GetAuthorId()
	post.Content = req.GetContent()
	post.Visibility = req.GetVisibility()

	return post, nil
}

func (s *fakeContentServiceServer) GetPost(ctx context.Context, req *contentpb.GetPostRequest) (*contentpb.PostResponse, error) {
	s.getPostCalled = true

	post := fakePostResponse()
	post.Id = req.GetPostId()

	return post, nil
}

func (s *fakeContentServiceServer) UpdatePost(ctx context.Context, req *contentpb.UpdatePostRequest) (*contentpb.PostResponse, error) {
	s.updatePostCalled = true

	post := fakePostResponse()
	post.Id = req.GetPostId()
	post.Content = req.GetContent()
	post.Visibility = req.GetVisibility()

	return post, nil
}

func (s *fakeContentServiceServer) DeletePost(ctx context.Context, req *contentpb.DeletePostRequest) (*contentpb.DeletePostResponse, error) {
	s.deletePostCalled = true

	return &contentpb.DeletePostResponse{
		Success: true,
		Message: "post deleted successfully",
	}, nil
}

func (s *fakeContentServiceServer) CreateReply(ctx context.Context, req *contentpb.CreateReplyRequest) (*contentpb.PostResponse, error) {
	s.createReplyCalled = true

	post := fakePostResponse()
	post.ReplyToPostId = req.GetReplyToPostId()
	post.Content = req.GetContent()

	return post, nil
}

func (s *fakeContentServiceServer) GetReplies(ctx context.Context, req *contentpb.GetRepliesRequest) (*contentpb.PostListResponse, error) {
	s.getRepliesCalled = true

	return &contentpb.PostListResponse{
		Posts: []*contentpb.PostResponse{
			fakePostResponse(),
		},
		NextCursor: "next-cursor",
	}, nil
}

func (s *fakeContentServiceServer) LikePost(ctx context.Context, req *contentpb.LikePostRequest) (*contentpb.ActionResponse, error) {
	s.likePostCalled = true
	return fakeActionResponse("post liked successfully"), nil
}

func (s *fakeContentServiceServer) UnlikePost(ctx context.Context, req *contentpb.UnlikePostRequest) (*contentpb.ActionResponse, error) {
	s.unlikePostCalled = true
	return fakeActionResponse("post unliked successfully"), nil
}

func (s *fakeContentServiceServer) BookmarkPost(ctx context.Context, req *contentpb.BookmarkPostRequest) (*contentpb.ActionResponse, error) {
	s.bookmarkPostCalled = true
	return fakeActionResponse("post bookmarked successfully"), nil
}

func (s *fakeContentServiceServer) UnbookmarkPost(ctx context.Context, req *contentpb.UnbookmarkPostRequest) (*contentpb.ActionResponse, error) {
	s.unbookmarkPostCalled = true
	return fakeActionResponse("post unbookmarked successfully"), nil
}

func (s *fakeContentServiceServer) RepostPost(ctx context.Context, req *contentpb.RepostPostRequest) (*contentpb.ActionResponse, error) {
	s.repostPostCalled = true
	return fakeActionResponse("post reposted successfully"), nil
}

func (s *fakeContentServiceServer) UndoRepost(ctx context.Context, req *contentpb.UndoRepostRequest) (*contentpb.ActionResponse, error) {
	s.undoRepostCalled = true
	return fakeActionResponse("repost removed successfully"), nil
}

func (s *fakeContentServiceServer) FollowUser(ctx context.Context, req *contentpb.FollowUserRequest) (*contentpb.ActionResponse, error) {
	s.followUserCalled = true
	return fakeActionResponse("user followed successfully"), nil
}

func (s *fakeContentServiceServer) UnfollowUser(ctx context.Context, req *contentpb.UnfollowUserRequest) (*contentpb.ActionResponse, error) {
	s.unfollowUserCalled = true
	return fakeActionResponse("user unfollowed successfully"), nil
}

func (s *fakeContentServiceServer) GetFollowers(ctx context.Context, req *contentpb.GetFollowersRequest) (*contentpb.UserListResponse, error) {
	s.getFollowersCalled = true

	return &contentpb.UserListResponse{
		Users: []*contentpb.UserSummary{
			{
				Id:          "user-1",
				Username:    "follower",
				DisplayName: "Follower User",
				IsVerified:  true,
			},
		},
	}, nil
}

func (s *fakeContentServiceServer) GetFollowing(ctx context.Context, req *contentpb.GetFollowingRequest) (*contentpb.UserListResponse, error) {
	s.getFollowingCalled = true

	return &contentpb.UserListResponse{
		Users: []*contentpb.UserSummary{
			{
				Id:          "user-2",
				Username:    "following",
				DisplayName: "Following User",
			},
		},
	}, nil
}

func (s *fakeContentServiceServer) GetUserTimeline(ctx context.Context, req *contentpb.GetUserTimelineRequest) (*contentpb.TimelineResponse, error) {
	s.getUserTimelineCalled = true

	return &contentpb.TimelineResponse{
		Items: []*contentpb.TimelineItemResponse{
			{
				Type: "post",
				Post: fakePostResponse(),
			},
			{
				Type:       "repost",
				Post:       fakePostResponse(),
				RepostedAt: "2026-07-02T11:00:00Z",
			},
		},
	}, nil
}

func TestContentClientGetHealth(t *testing.T) {
	fakeServer := &fakeContentServiceServer{}
	client, cleanup := startFakeContentGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.GetHealth(context.Background())

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.healthCalled)
	assert.Equal(t, "ok", res.Status)
	assert.Equal(t, "content-service", res.Service)
}

func TestContentClientCreatePost(t *testing.T) {
	fakeServer := &fakeContentServiceServer{}
	client, cleanup := startFakeContentGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.CreatePost(context.Background(), &contentpb.CreatePostRequest{
		AuthorId:   "author-1",
		Content:    "hello",
		Visibility: "public",
		MediaIds:   []string{"media-1"},
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.createPostCalled)
	assert.Equal(t, "author-1", res.AuthorId)
	assert.Equal(t, "hello", res.Content)
}

func TestContentClientGetPost(t *testing.T) {
	fakeServer := &fakeContentServiceServer{}
	client, cleanup := startFakeContentGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.GetPost(context.Background(), &contentpb.GetPostRequest{
		PostId: "post-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.getPostCalled)
	assert.Equal(t, "post-1", res.Id)
}

func TestContentClientUpdatePost(t *testing.T) {
	fakeServer := &fakeContentServiceServer{}
	client, cleanup := startFakeContentGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.UpdatePost(context.Background(), &contentpb.UpdatePostRequest{
		PostId:     "post-1",
		Content:    "updated",
		Visibility: "followers",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.updatePostCalled)
	assert.Equal(t, "updated", res.Content)
	assert.Equal(t, "followers", res.Visibility)
}

func TestContentClientDeletePost(t *testing.T) {
	fakeServer := &fakeContentServiceServer{}
	client, cleanup := startFakeContentGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.DeletePost(context.Background(), &contentpb.DeletePostRequest{
		PostId:      "post-1",
		RequesterId: "user-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.deletePostCalled)
	assert.True(t, res.Success)
}

func TestContentClientCreateReply(t *testing.T) {
	fakeServer := &fakeContentServiceServer{}
	client, cleanup := startFakeContentGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.CreateReply(context.Background(), &contentpb.CreateReplyRequest{
		AuthorId:      "author-1",
		ReplyToPostId: "parent-1",
		Content:       "reply",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.createReplyCalled)
	assert.Equal(t, "parent-1", res.ReplyToPostId)
	assert.Equal(t, "reply", res.Content)
}

func TestContentClientGetReplies(t *testing.T) {
	fakeServer := &fakeContentServiceServer{}
	client, cleanup := startFakeContentGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.GetReplies(context.Background(), &contentpb.GetRepliesRequest{
		PostId: "post-1",
		Limit:  10,
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.getRepliesCalled)
	require.Len(t, res.Posts, 1)
	assert.Equal(t, "next-cursor", res.NextCursor)
}

func TestContentClientPostActions(t *testing.T) {
	tests := []struct {
		name    string
		call    func(client *ContentClient) (*contentpb.ActionResponse, error)
		assert  func(fakeServer *fakeContentServiceServer)
		message string
	}{
		{
			name: "like post",
			call: func(client *ContentClient) (*contentpb.ActionResponse, error) {
				return client.LikePost(context.Background(), &contentpb.LikePostRequest{
					UserId: "user-1",
					PostId: "post-1",
				})
			},
			assert: func(fakeServer *fakeContentServiceServer) {
				assert.True(t, fakeServer.likePostCalled)
			},
			message: "post liked successfully",
		},
		{
			name: "unlike post",
			call: func(client *ContentClient) (*contentpb.ActionResponse, error) {
				return client.UnlikePost(context.Background(), &contentpb.UnlikePostRequest{
					UserId: "user-1",
					PostId: "post-1",
				})
			},
			assert: func(fakeServer *fakeContentServiceServer) {
				assert.True(t, fakeServer.unlikePostCalled)
			},
			message: "post unliked successfully",
		},
		{
			name: "bookmark post",
			call: func(client *ContentClient) (*contentpb.ActionResponse, error) {
				return client.BookmarkPost(context.Background(), &contentpb.BookmarkPostRequest{
					UserId: "user-1",
					PostId: "post-1",
				})
			},
			assert: func(fakeServer *fakeContentServiceServer) {
				assert.True(t, fakeServer.bookmarkPostCalled)
			},
			message: "post bookmarked successfully",
		},
		{
			name: "unbookmark post",
			call: func(client *ContentClient) (*contentpb.ActionResponse, error) {
				return client.UnbookmarkPost(context.Background(), &contentpb.UnbookmarkPostRequest{
					UserId: "user-1",
					PostId: "post-1",
				})
			},
			assert: func(fakeServer *fakeContentServiceServer) {
				assert.True(t, fakeServer.unbookmarkPostCalled)
			},
			message: "post unbookmarked successfully",
		},
		{
			name: "repost post",
			call: func(client *ContentClient) (*contentpb.ActionResponse, error) {
				return client.RepostPost(context.Background(), &contentpb.RepostPostRequest{
					UserId: "user-1",
					PostId: "post-1",
				})
			},
			assert: func(fakeServer *fakeContentServiceServer) {
				assert.True(t, fakeServer.repostPostCalled)
			},
			message: "post reposted successfully",
		},
		{
			name: "undo repost",
			call: func(client *ContentClient) (*contentpb.ActionResponse, error) {
				return client.UndoRepost(context.Background(), &contentpb.UndoRepostRequest{
					UserId: "user-1",
					PostId: "post-1",
				})
			},
			assert: func(fakeServer *fakeContentServiceServer) {
				assert.True(t, fakeServer.undoRepostCalled)
			},
			message: "repost removed successfully",
		},
		{
			name: "follow user",
			call: func(client *ContentClient) (*contentpb.ActionResponse, error) {
				return client.FollowUser(context.Background(), &contentpb.FollowUserRequest{
					FollowerId:  "user-1",
					FollowingId: "user-2",
				})
			},
			assert: func(fakeServer *fakeContentServiceServer) {
				assert.True(t, fakeServer.followUserCalled)
			},
			message: "user followed successfully",
		},
		{
			name: "unfollow user",
			call: func(client *ContentClient) (*contentpb.ActionResponse, error) {
				return client.UnfollowUser(context.Background(), &contentpb.UnfollowUserRequest{
					FollowerId:  "user-1",
					FollowingId: "user-2",
				})
			},
			assert: func(fakeServer *fakeContentServiceServer) {
				assert.True(t, fakeServer.unfollowUserCalled)
			},
			message: "user unfollowed successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeServer := &fakeContentServiceServer{}
			client, cleanup := startFakeContentGRPCServer(t, fakeServer)
			defer cleanup()

			res, err := tt.call(client)

			require.NoError(t, err)
			require.NotNil(t, res)

			tt.assert(fakeServer)
			assert.True(t, res.Success)
			assert.Equal(t, tt.message, res.Message)
		})
	}
}

func TestContentClientGetFollowers(t *testing.T) {
	fakeServer := &fakeContentServiceServer{}
	client, cleanup := startFakeContentGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.GetFollowers(context.Background(), &contentpb.GetFollowersRequest{
		UserId: "user-1",
		Limit:  10,
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.getFollowersCalled)
	require.Len(t, res.Users, 1)
	assert.Equal(t, "follower", res.Users[0].Username)
}

func TestContentClientGetFollowing(t *testing.T) {
	fakeServer := &fakeContentServiceServer{}
	client, cleanup := startFakeContentGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.GetFollowing(context.Background(), &contentpb.GetFollowingRequest{
		UserId: "user-1",
		Limit:  10,
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.getFollowingCalled)
	require.Len(t, res.Users, 1)
	assert.Equal(t, "following", res.Users[0].Username)
}

func TestContentClientGetUserTimeline(t *testing.T) {
	fakeServer := &fakeContentServiceServer{}
	client, cleanup := startFakeContentGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.GetUserTimeline(context.Background(), &contentpb.GetUserTimelineRequest{
		UserId: "user-1",
		Limit:  10,
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.getUserTimelineCalled)
	require.Len(t, res.Items, 2)
	assert.Equal(t, "post", res.Items[0].Type)
	assert.Equal(t, "repost", res.Items[1].Type)
}
