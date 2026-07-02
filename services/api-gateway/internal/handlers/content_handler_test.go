package handlers

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	"github.com/trungquantrannguyen/threadly/pkg/middleware"
	contentpb "github.com/trungquantrannguyen/threadly/proto/content"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeContentHandlerServiceServer struct {
	contentpb.UnimplementedContentServiceServer

	err error

	healthCalled       bool
	createPostCalled   bool
	getPostCalled      bool
	updatePostCalled   bool
	deletePostCalled   bool
	createReplyCalled  bool
	getRepliesCalled   bool
	actionCalled       string
	getFollowersCalled bool
	getFollowingCalled bool

	receivedAuthorID    string
	receivedRequesterID string
	receivedUserID      string
	receivedPostID      string
	receivedTargetID    string
	receivedContent     string
	receivedVisibility  string
	receivedLimit       int32
	receivedCursor      string
}

func startContentHandlerTestClient(t *testing.T, fakeServer *fakeContentHandlerServiceServer) (*client.ContentClient, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	contentpb.RegisterContentServiceServer(grpcServer, fakeServer)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	contentClient, err := client.NewContentClient(config.Config{
		ContentServiceGRPCAddr: listener.Addr().String(),
	}, zerolog.Nop())
	require.NoError(t, err)

	cleanup := func() {
		_ = contentClient.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}

	return contentClient, cleanup
}

func fakeGatewayContentPost() *contentpb.PostResponse {
	return &contentpb.PostResponse{
		Id:            "post-1",
		AuthorId:      "author-1",
		ReplyToPostId: "",
		Content:       "hello content",
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

func fakeGatewayAction(message string) *contentpb.ActionResponse {
	return &contentpb.ActionResponse{
		Success: true,
		Message: message,
	}
}

func (s *fakeContentHandlerServiceServer) GetHealth(ctx context.Context, req *contentpb.GetContentServiceHealthRequest) (*contentpb.GetContentServiceHealthResponse, error) {
	s.healthCalled = true

	if s.err != nil {
		return nil, s.err
	}

	return &contentpb.GetContentServiceHealthResponse{
		Service:   "content-service",
		Status:    "ok",
		Env:       "test",
		CheckedAt: "now",
	}, nil
}

func (s *fakeContentHandlerServiceServer) CreatePost(ctx context.Context, req *contentpb.CreatePostRequest) (*contentpb.PostResponse, error) {
	s.createPostCalled = true
	s.receivedAuthorID = req.GetAuthorId()
	s.receivedContent = req.GetContent()
	s.receivedVisibility = req.GetVisibility()

	if s.err != nil {
		return nil, s.err
	}

	post := fakeGatewayContentPost()
	post.AuthorId = req.GetAuthorId()
	post.Content = req.GetContent()
	post.Visibility = req.GetVisibility()

	return post, nil
}

func (s *fakeContentHandlerServiceServer) GetPost(ctx context.Context, req *contentpb.GetPostRequest) (*contentpb.PostResponse, error) {
	s.getPostCalled = true
	s.receivedPostID = req.GetPostId()

	if s.err != nil {
		return nil, s.err
	}

	post := fakeGatewayContentPost()
	post.Id = req.GetPostId()

	return post, nil
}

func (s *fakeContentHandlerServiceServer) UpdatePost(ctx context.Context, req *contentpb.UpdatePostRequest) (*contentpb.PostResponse, error) {
	s.updatePostCalled = true
	s.receivedPostID = req.GetPostId()
	s.receivedRequesterID = req.GetRequesterId()
	s.receivedContent = req.GetContent()
	s.receivedVisibility = req.GetVisibility()

	if s.err != nil {
		return nil, s.err
	}

	post := fakeGatewayContentPost()
	post.Id = req.GetPostId()
	post.Content = req.GetContent()
	post.Visibility = req.GetVisibility()

	return post, nil
}

func (s *fakeContentHandlerServiceServer) DeletePost(ctx context.Context, req *contentpb.DeletePostRequest) (*contentpb.DeletePostResponse, error) {
	s.deletePostCalled = true
	s.receivedPostID = req.GetPostId()
	s.receivedRequesterID = req.GetRequesterId()

	if s.err != nil {
		return nil, s.err
	}

	return &contentpb.DeletePostResponse{
		Success: true,
		Message: "post deleted successfully",
	}, nil
}

func (s *fakeContentHandlerServiceServer) CreateReply(ctx context.Context, req *contentpb.CreateReplyRequest) (*contentpb.PostResponse, error) {
	s.createReplyCalled = true
	s.receivedAuthorID = req.GetAuthorId()
	s.receivedPostID = req.GetReplyToPostId()
	s.receivedContent = req.GetContent()

	if s.err != nil {
		return nil, s.err
	}

	reply := fakeGatewayContentPost()
	reply.Id = "reply-1"
	reply.ReplyToPostId = req.GetReplyToPostId()
	reply.Content = req.GetContent()

	return reply, nil
}

func (s *fakeContentHandlerServiceServer) GetReplies(ctx context.Context, req *contentpb.GetRepliesRequest) (*contentpb.PostListResponse, error) {
	s.getRepliesCalled = true
	s.receivedPostID = req.GetPostId()
	s.receivedLimit = req.GetLimit()
	s.receivedCursor = req.GetCursor()

	if s.err != nil {
		return nil, s.err
	}

	reply := fakeGatewayContentPost()
	reply.Id = "reply-1"
	reply.ReplyToPostId = req.GetPostId()

	return &contentpb.PostListResponse{
		Posts: []*contentpb.PostResponse{reply},
	}, nil
}

func (s *fakeContentHandlerServiceServer) LikePost(ctx context.Context, req *contentpb.LikePostRequest) (*contentpb.ActionResponse, error) {
	s.actionCalled = "like"
	s.receivedUserID = req.GetUserId()
	s.receivedPostID = req.GetPostId()

	if s.err != nil {
		return nil, s.err
	}

	return fakeGatewayAction("post liked successfully"), nil
}

func (s *fakeContentHandlerServiceServer) UnlikePost(ctx context.Context, req *contentpb.UnlikePostRequest) (*contentpb.ActionResponse, error) {
	s.actionCalled = "unlike"
	s.receivedUserID = req.GetUserId()
	s.receivedPostID = req.GetPostId()

	if s.err != nil {
		return nil, s.err
	}

	return fakeGatewayAction("post unliked successfully"), nil
}

func (s *fakeContentHandlerServiceServer) BookmarkPost(ctx context.Context, req *contentpb.BookmarkPostRequest) (*contentpb.ActionResponse, error) {
	s.actionCalled = "bookmark"
	s.receivedUserID = req.GetUserId()
	s.receivedPostID = req.GetPostId()

	if s.err != nil {
		return nil, s.err
	}

	return fakeGatewayAction("post bookmarked successfully"), nil
}

func (s *fakeContentHandlerServiceServer) UnbookmarkPost(ctx context.Context, req *contentpb.UnbookmarkPostRequest) (*contentpb.ActionResponse, error) {
	s.actionCalled = "unbookmark"
	s.receivedUserID = req.GetUserId()
	s.receivedPostID = req.GetPostId()

	if s.err != nil {
		return nil, s.err
	}

	return fakeGatewayAction("post unbookmarked successfully"), nil
}

func (s *fakeContentHandlerServiceServer) RepostPost(ctx context.Context, req *contentpb.RepostPostRequest) (*contentpb.ActionResponse, error) {
	s.actionCalled = "repost"
	s.receivedUserID = req.GetUserId()
	s.receivedPostID = req.GetPostId()

	if s.err != nil {
		return nil, s.err
	}

	return fakeGatewayAction("post reposted successfully"), nil
}

func (s *fakeContentHandlerServiceServer) UndoRepost(ctx context.Context, req *contentpb.UndoRepostRequest) (*contentpb.ActionResponse, error) {
	s.actionCalled = "undo_repost"
	s.receivedUserID = req.GetUserId()
	s.receivedPostID = req.GetPostId()

	if s.err != nil {
		return nil, s.err
	}

	return fakeGatewayAction("repost removed successfully"), nil
}

func (s *fakeContentHandlerServiceServer) FollowUser(ctx context.Context, req *contentpb.FollowUserRequest) (*contentpb.ActionResponse, error) {
	s.actionCalled = "follow"
	s.receivedUserID = req.GetFollowerId()
	s.receivedTargetID = req.GetFollowingId()

	if s.err != nil {
		return nil, s.err
	}

	return fakeGatewayAction("user followed successfully"), nil
}

func (s *fakeContentHandlerServiceServer) UnfollowUser(ctx context.Context, req *contentpb.UnfollowUserRequest) (*contentpb.ActionResponse, error) {
	s.actionCalled = "unfollow"
	s.receivedUserID = req.GetFollowerId()
	s.receivedTargetID = req.GetFollowingId()

	if s.err != nil {
		return nil, s.err
	}

	return fakeGatewayAction("user unfollowed successfully"), nil
}

func (s *fakeContentHandlerServiceServer) GetFollowers(ctx context.Context, req *contentpb.GetFollowersRequest) (*contentpb.UserListResponse, error) {
	s.getFollowersCalled = true
	s.receivedUserID = req.GetUserId()
	s.receivedLimit = req.GetLimit()
	s.receivedCursor = req.GetCursor()

	if s.err != nil {
		return nil, s.err
	}

	return &contentpb.UserListResponse{
		Users: []*contentpb.UserSummary{
			{
				Id:          "follower-1",
				Username:    "follower",
				DisplayName: "Follower User",
				IsVerified:  true,
			},
		},
	}, nil
}

func (s *fakeContentHandlerServiceServer) GetFollowing(ctx context.Context, req *contentpb.GetFollowingRequest) (*contentpb.UserListResponse, error) {
	s.getFollowingCalled = true
	s.receivedUserID = req.GetUserId()
	s.receivedLimit = req.GetLimit()
	s.receivedCursor = req.GetCursor()

	if s.err != nil {
		return nil, s.err
	}

	return &contentpb.UserListResponse{
		Users: []*contentpb.UserSummary{
			{
				Id:          "following-1",
				Username:    "following",
				DisplayName: "Following User",
			},
		},
	}, nil
}

func newContentHandlerTestRouter(handler *ContentHandler, withUser bool) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	if withUser {
		router.Use(func(c *gin.Context) {
			c.Set(middleware.UserIDKey, "user-1")
			c.Next()
		})
	}

	router.GET("/contents/health", handler.GetHealth)

	posts := router.Group("/contents/posts")
	{
		posts.POST("", handler.CreatePost)
		posts.GET("/:postID", handler.GetPost)
		posts.PATCH("/:postID", handler.UpdatePost)
		posts.DELETE("/:postID", handler.DeletePost)

		posts.POST("/:postID/replies", handler.CreateReply)
		posts.GET("/:postID/replies", handler.GetReplies)

		posts.POST("/:postID/likes", handler.LikePost)
		posts.DELETE("/:postID/likes", handler.UnlikePost)

		posts.POST("/:postID/bookmarks", handler.BookmarkPost)
		posts.DELETE("/:postID/bookmarks", handler.UnbookmarkPost)

		posts.POST("/:postID/reposts", handler.RepostPost)
		posts.DELETE("/:postID/reposts", handler.UndoRepost)
	}

	users := router.Group("/contents/users")
	{
		users.POST("/:userID/follow", handler.FollowUser)
		users.DELETE("/:userID/follow", handler.UnfollowUser)
		users.GET("/:userID/followers", handler.GetFollowers)
		users.GET("/:userID/following", handler.GetFollowing)
	}

	return router
}

func newContentHandler(t *testing.T, fakeServer *fakeContentHandlerServiceServer) (*ContentHandler, func()) {
	t.Helper()

	contentClient, cleanup := startContentHandlerTestClient(t, fakeServer)
	handler := NewContentHandler(contentClient, zerolog.Nop())

	return handler, cleanup
}

func TestContentHandlerGetHealthSuccess(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contents/health", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.healthCalled)
	assert.Contains(t, w.Body.String(), "Content service available")
	assert.Contains(t, w.Body.String(), "content-service")
}

func TestContentHandlerGetHealthServiceUnavailable(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{
		err: errors.New("content service down"),
	}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contents/health", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.True(t, fakeServer.healthCalled)
	assert.Contains(t, w.Body.String(), "Content service unavailable")
}

func TestContentHandlerCreatePostSuccess(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	body := bytes.NewBufferString(`{
		"content": "hello content",
		"visibility": "public",
		"media_ids": ["media-1"]
	}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/contents/posts", body)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.True(t, fakeServer.createPostCalled)
	assert.Equal(t, "user-1", fakeServer.receivedAuthorID)
	assert.Equal(t, "hello content", fakeServer.receivedContent)
	assert.Equal(t, "public", fakeServer.receivedVisibility)
	assert.Contains(t, w.Body.String(), "Created a post successfully")
}

func TestContentHandlerCreatePostInvalidBody(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/contents/posts", bytes.NewBufferString(`bad-json`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, fakeServer.createPostCalled)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

func TestContentHandlerCreatePostUnauthorized(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/contents/posts", bytes.NewBufferString(`{"content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, fakeServer.createPostCalled)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestContentHandlerCreatePostGRPCError(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{
		err: status.Error(codes.InvalidArgument, "invalid content"),
	}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/contents/posts", bytes.NewBufferString(`{"content":"bad"}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.True(t, fakeServer.createPostCalled)
	assert.Contains(t, w.Body.String(), "invalid content")
}

func TestContentHandlerGetPostSuccess(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contents/posts/post-1", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.getPostCalled)
	assert.Equal(t, "post-1", fakeServer.receivedPostID)
	assert.Contains(t, w.Body.String(), "Get post successfully")
	assert.Contains(t, w.Body.String(), "post-1")
}

func TestContentHandlerGetPostNotFound(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{
		err: status.Error(codes.NotFound, "post not found"),
	}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contents/posts/post-1", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.True(t, fakeServer.getPostCalled)
	assert.Contains(t, w.Body.String(), "post not found")
}

func TestContentHandlerUpdatePostSuccess(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	body := bytes.NewBufferString(`{
		"content": "updated content",
		"visibility": "followers",
		"media_ids": ["media-1"]
	}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/contents/posts/post-1", body)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.updatePostCalled)
	assert.Equal(t, "post-1", fakeServer.receivedPostID)
	assert.Equal(t, "user-1", fakeServer.receivedRequesterID)
	assert.Equal(t, "updated content", fakeServer.receivedContent)
	assert.Equal(t, "followers", fakeServer.receivedVisibility)
	assert.Contains(t, w.Body.String(), "Updated post successfully")
}

func TestContentHandlerUpdatePostInvalidBody(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/contents/posts/post-1", bytes.NewBufferString(`bad-json`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, fakeServer.updatePostCalled)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

func TestContentHandlerUpdatePostUnauthorized(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/contents/posts/post-1", bytes.NewBufferString(`{"content":"updated"}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, fakeServer.updatePostCalled)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestContentHandlerDeletePostSuccess(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/contents/posts/post-1", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, fakeServer.deletePostCalled)
	assert.Equal(t, "post-1", fakeServer.receivedPostID)
	assert.Equal(t, "user-1", fakeServer.receivedRequesterID)
}

func TestContentHandlerCreateReplySuccess(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/contents/posts/post-1/replies", bytes.NewBufferString(`{"content":"reply"}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.True(t, fakeServer.createReplyCalled)
	assert.Equal(t, "user-1", fakeServer.receivedAuthorID)
	assert.Equal(t, "post-1", fakeServer.receivedPostID)
	assert.Equal(t, "reply", fakeServer.receivedContent)
	assert.Contains(t, w.Body.String(), "Create reply successfully")
}

func TestContentHandlerCreateReplyUnauthorized(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/contents/posts/post-1/replies", bytes.NewBufferString(`{"content":"reply"}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, fakeServer.createReplyCalled)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestContentHandlerGetRepliesSuccess(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contents/posts/post-1/replies?limit=10&cursor=abc", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.getRepliesCalled)
	assert.Equal(t, "post-1", fakeServer.receivedPostID)
	assert.Equal(t, int32(10), fakeServer.receivedLimit)
	assert.Equal(t, "abc", fakeServer.receivedCursor)
	assert.Contains(t, w.Body.String(), "Get replies successfully")
	assert.Contains(t, w.Body.String(), "reply-1")
}

func TestContentHandlerGetRepliesInvalidLimit(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contents/posts/post-1/replies?limit=abc", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, fakeServer.getRepliesCalled)
	assert.Contains(t, w.Body.String(), "Invalid limit")
}

func TestContentHandlerPostActionsSuccess(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		expectedAction string
		expectedText   string
	}{
		{
			name:           "like",
			method:         http.MethodPost,
			path:           "/contents/posts/post-1/likes",
			expectedAction: "like",
			expectedText:   "post liked successfully",
		},
		{
			name:           "unlike",
			method:         http.MethodDelete,
			path:           "/contents/posts/post-1/likes",
			expectedAction: "unlike",
			expectedText:   "post unliked successfully",
		},
		{
			name:           "bookmark",
			method:         http.MethodPost,
			path:           "/contents/posts/post-1/bookmarks",
			expectedAction: "bookmark",
			expectedText:   "post bookmarked successfully",
		},
		{
			name:           "unbookmark",
			method:         http.MethodDelete,
			path:           "/contents/posts/post-1/bookmarks",
			expectedAction: "unbookmark",
			expectedText:   "post unbookmarked successfully",
		},
		{
			name:           "repost",
			method:         http.MethodPost,
			path:           "/contents/posts/post-1/reposts",
			expectedAction: "repost",
			expectedText:   "post reposted successfully",
		},
		{
			name:           "undo repost",
			method:         http.MethodDelete,
			path:           "/contents/posts/post-1/reposts",
			expectedAction: "undo_repost",
			expectedText:   "repost removed successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeServer := &fakeContentHandlerServiceServer{}
			handler, cleanup := newContentHandler(t, fakeServer)
			defer cleanup()

			router := newContentHandlerTestRouter(handler, true)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)

			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.expectedAction, fakeServer.actionCalled)
			assert.Equal(t, "user-1", fakeServer.receivedUserID)
			assert.Equal(t, "post-1", fakeServer.receivedPostID)
			assert.Contains(t, w.Body.String(), tt.expectedText)
		})
	}
}

func TestContentHandlerPostActionUnauthorized(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/contents/posts/post-1/likes", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, fakeServer.actionCalled)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestContentHandlerPostActionGRPCError(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{
		err: status.Error(codes.NotFound, "post not found"),
	}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/contents/posts/post-1/likes", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "like", fakeServer.actionCalled)
	assert.Contains(t, w.Body.String(), "post not found")
}

func TestContentHandlerUserActionsSuccess(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		expectedAction string
		expectedText   string
	}{
		{
			name:           "follow",
			method:         http.MethodPost,
			path:           "/contents/users/user-2/follow",
			expectedAction: "follow",
			expectedText:   "user followed successfully",
		},
		{
			name:           "unfollow",
			method:         http.MethodDelete,
			path:           "/contents/users/user-2/follow",
			expectedAction: "unfollow",
			expectedText:   "user unfollowed successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeServer := &fakeContentHandlerServiceServer{}
			handler, cleanup := newContentHandler(t, fakeServer)
			defer cleanup()

			router := newContentHandlerTestRouter(handler, true)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)

			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.expectedAction, fakeServer.actionCalled)
			assert.Equal(t, "user-1", fakeServer.receivedUserID)
			assert.Equal(t, "user-2", fakeServer.receivedTargetID)
			assert.Contains(t, w.Body.String(), tt.expectedText)
		})
	}
}

func TestContentHandlerUserActionUnauthorized(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/contents/users/user-2/follow", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, fakeServer.actionCalled)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestContentHandlerGetFollowersSuccess(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contents/users/user-2/followers?limit=10&cursor=abc", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.getFollowersCalled)
	assert.Equal(t, "user-2", fakeServer.receivedUserID)
	assert.Equal(t, int32(10), fakeServer.receivedLimit)
	assert.Equal(t, "abc", fakeServer.receivedCursor)
	assert.Contains(t, w.Body.String(), "Get followers successfully")
	assert.Contains(t, w.Body.String(), "follower")
}

func TestContentHandlerGetFollowingSuccess(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contents/users/user-2/following?limit=10&cursor=abc", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.getFollowingCalled)
	assert.Equal(t, "user-2", fakeServer.receivedUserID)
	assert.Equal(t, int32(10), fakeServer.receivedLimit)
	assert.Equal(t, "abc", fakeServer.receivedCursor)
	assert.Contains(t, w.Body.String(), "Get following successfully")
	assert.Contains(t, w.Body.String(), "following")
}

func TestContentHandlerGetFollowersInvalidLimit(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contents/users/user-2/followers?limit=999", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, fakeServer.getFollowersCalled)
	assert.Contains(t, w.Body.String(), "Invalid limit")
}

func TestContentHandlerGetFollowingGRPCError(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{
		err: status.Error(codes.NotFound, "user not found"),
	}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contents/users/user-2/following", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.True(t, fakeServer.getFollowingCalled)
	assert.Contains(t, w.Body.String(), "user not found")
}

func TestContentHandlerGetPostMissingPostID(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contents/posts/", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.False(t, fakeServer.getPostCalled)
}

func TestContentHandlerUpdatePostGRPCError(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{
		err: status.Error(codes.PermissionDenied, "cannot update this post"),
	}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPatch,
		"/contents/posts/post-1",
		bytes.NewBufferString(`{"content":"updated","visibility":"public"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.True(t, fakeServer.updatePostCalled)
	assert.Contains(t, w.Body.String(), "cannot update this post")
}

func TestContentHandlerDeletePostUnauthorized(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/contents/posts/post-1", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, fakeServer.deletePostCalled)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestContentHandlerDeletePostGRPCError(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{
		err: status.Error(codes.PermissionDenied, "cannot delete this post"),
	}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/contents/posts/post-1", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.True(t, fakeServer.deletePostCalled)
	assert.Contains(t, w.Body.String(), "cannot delete this post")
}

func TestContentHandlerCreateReplyInvalidBody(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/contents/posts/post-1/replies", bytes.NewBufferString(`bad-json`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, fakeServer.createReplyCalled)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

func TestContentHandlerCreateReplyGRPCError(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{
		err: status.Error(codes.NotFound, "parent post not found"),
	}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/contents/posts/post-1/replies", bytes.NewBufferString(`{"content":"reply"}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.True(t, fakeServer.createReplyCalled)
	assert.Contains(t, w.Body.String(), "parent post not found")
}

func TestContentHandlerGetRepliesGRPCError(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{
		err: status.Error(codes.Internal, "failed to get replies"),
	}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contents/posts/post-1/replies", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.True(t, fakeServer.getRepliesCalled)
	assert.Contains(t, w.Body.String(), "Internal server error")
}

func TestContentHandlerUnbookmarkPostGRPCError(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{
		err: status.Error(codes.NotFound, "bookmark not found"),
	}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/contents/posts/post-1/bookmarks", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "unbookmark", fakeServer.actionCalled)
	assert.Contains(t, w.Body.String(), "bookmark not found")
}

func TestContentHandlerUndoRepostGRPCError(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{
		err: status.Error(codes.NotFound, "repost not found"),
	}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/contents/posts/post-1/reposts", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "undo_repost", fakeServer.actionCalled)
	assert.Contains(t, w.Body.String(), "repost not found")
}

func TestContentHandlerUnfollowUserGRPCError(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{
		err: status.Error(codes.NotFound, "follow relationship not found"),
	}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/contents/users/user-2/follow", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "unfollow", fakeServer.actionCalled)
	assert.Contains(t, w.Body.String(), "follow relationship not found")
}

func TestContentHandlerGetFollowersGRPCError(t *testing.T) {
	fakeServer := &fakeContentHandlerServiceServer{
		err: status.Error(codes.NotFound, "user not found"),
	}
	handler, cleanup := newContentHandler(t, fakeServer)
	defer cleanup()

	router := newContentHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contents/users/user-2/followers", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.True(t, fakeServer.getFollowersCalled)
	assert.Contains(t, w.Body.String(), "user not found")
}
