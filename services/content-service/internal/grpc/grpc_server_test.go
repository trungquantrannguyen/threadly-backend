package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	contentpb "github.com/trungquantrannguyen/threadly/proto/content"
	"github.com/trungquantrannguyen/threadly/services/content-service/internal/dto"
	"github.com/trungquantrannguyen/threadly/services/content-service/internal/repository"
	"github.com/trungquantrannguyen/threadly/services/content-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeContentService struct {
	post      *dto.PostResponse
	posts     []dto.PostResponse
	action    *dto.ActionResponse
	users     []dto.UserSummary
	timeline  []dto.TimelineItemResponse
	err       error
	deleteErr error

	createPostReq   dto.CreatePostRequest
	getPostReq      dto.GetPostRequest
	updatePostReq   dto.UpdatePostRequest
	deletePostReq   dto.DeletePostRequest
	createReplyReq  dto.CreateReplyRequest
	getRepliesReq   dto.GetRepliesRequest
	likeReq         dto.LikePostRequest
	unlikeReq       dto.UnlikePostRequest
	bookmarkReq     dto.BookmarkPostRequest
	unbookmarkReq   dto.UnbookmarkPostRequest
	repostReq       dto.RepostPostRequest
	undoRepostReq   dto.UndoRepostRequest
	followReq       dto.FollowUserRequest
	unfollowReq     dto.UnfollowUserRequest
	getFollowersReq dto.GetFollowersRequest
	getFollowingReq dto.GetFollowingRequest
	timelineReq     dto.GetUserTimelineRequest
}

func (f *fakeContentService) CreatePost(ctx context.Context, req dto.CreatePostRequest) (*dto.PostResponse, error) {
	f.createPostReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.post, nil
}

func (f *fakeContentService) GetPost(ctx context.Context, req dto.GetPostRequest) (*dto.PostResponse, error) {
	f.getPostReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.post, nil
}

func (f *fakeContentService) UpdatePost(ctx context.Context, req dto.UpdatePostRequest) (*dto.PostResponse, error) {
	f.updatePostReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.post, nil
}

func (f *fakeContentService) DeletePost(ctx context.Context, req dto.DeletePostRequest) error {
	f.deletePostReq = req
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeContentService) CreateReply(ctx context.Context, req dto.CreateReplyRequest) (*dto.PostResponse, error) {
	f.createReplyReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.post, nil
}

func (f *fakeContentService) GetReplies(ctx context.Context, req dto.GetRepliesRequest) ([]dto.PostResponse, error) {
	f.getRepliesReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.posts, nil
}

func (f *fakeContentService) LikePost(ctx context.Context, req dto.LikePostRequest) (*dto.ActionResponse, error) {
	f.likeReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.action, nil
}

func (f *fakeContentService) UnlikePost(ctx context.Context, req dto.UnlikePostRequest) (*dto.ActionResponse, error) {
	f.unlikeReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.action, nil
}

func (f *fakeContentService) BookmarkPost(ctx context.Context, req dto.BookmarkPostRequest) (*dto.ActionResponse, error) {
	f.bookmarkReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.action, nil
}

func (f *fakeContentService) UnbookmarkPost(ctx context.Context, req dto.UnbookmarkPostRequest) (*dto.ActionResponse, error) {
	f.unbookmarkReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.action, nil
}

func (f *fakeContentService) RepostPost(ctx context.Context, req dto.RepostPostRequest) (*dto.ActionResponse, error) {
	f.repostReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.action, nil
}

func (f *fakeContentService) UndoRepost(ctx context.Context, req dto.UndoRepostRequest) (*dto.ActionResponse, error) {
	f.undoRepostReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.action, nil
}

func (f *fakeContentService) FollowUser(ctx context.Context, req dto.FollowUserRequest) (*dto.ActionResponse, error) {
	f.followReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.action, nil
}

func (f *fakeContentService) UnfollowUser(ctx context.Context, req dto.UnfollowUserRequest) (*dto.ActionResponse, error) {
	f.unfollowReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.action, nil
}

func (f *fakeContentService) GetFollowers(ctx context.Context, req dto.GetFollowersRequest) ([]dto.UserSummary, error) {
	f.getFollowersReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.users, nil
}

func (f *fakeContentService) GetFollowing(ctx context.Context, req dto.GetFollowingRequest) ([]dto.UserSummary, error) {
	f.getFollowingReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.users, nil
}

func (f *fakeContentService) GetUserTimeline(ctx context.Context, req dto.GetUserTimelineRequest) ([]dto.TimelineItemResponse, error) {
	f.timelineReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.timeline, nil
}

func newTestContentServer(fakeSvc *fakeContentService) *ContentServiceServer {
	return NewContentServiceServer(
		config.Config{
			ServiceName: "content-service",
			AppEnv:      "test",
		},
		zerolog.Nop(),
		fakeSvc,
	)
}

func testPostResponse() *dto.PostResponse {
	return &dto.PostResponse{
		ID:            "post-1",
		AuthorID:      "author-1",
		ReplyToPostID: "",
		Content:       "hello",
		Visibility:    "public",
		LikeCount:     1,
		ReplyCount:    2,
		RepostCount:   3,
		BookmarkCount: 4,
		CreatedAt:     "2026-07-02T10:00:00Z",
		UpdatedAt:     "2026-07-02T10:01:00Z",
		Author: dto.UserSummary{
			ID:          "author-1",
			Username:    "author",
			DisplayName: "Author User",
			AvatarURL:   "https://example.com/avatar.png",
			IsVerified:  true,
		},
		Media: []dto.MediaResponse{
			{
				ID:        "media-1",
				URL:       "https://example.com/media.png",
				MimeType:  "image/png",
				SizeBytes: 1024,
				Width:     640,
				Height:    480,
			},
		},
	}
}

func testActionResponse(message string) *dto.ActionResponse {
	return &dto.ActionResponse{
		Success: true,
		Message: message,
	}
}

func TestGetHealth(t *testing.T) {
	server := newTestContentServer(&fakeContentService{})

	res, err := server.GetHealth(context.Background(), &contentpb.GetContentServiceHealthRequest{})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "ok", res.Status)
	assert.Equal(t, "content-service", res.Service)
	assert.Equal(t, "test", res.Env)
	assert.NotEmpty(t, res.CheckedAt)
}

func TestCreatePostSuccess(t *testing.T) {
	fakeSvc := &fakeContentService{
		post: testPostResponse(),
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.CreatePost(context.Background(), &contentpb.CreatePostRequest{
		AuthorId:   "author-1",
		Content:    "hello",
		Visibility: "public",
		MediaIds:   []string{"media-1"},
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "author-1", fakeSvc.createPostReq.AuthorID)
	assert.Equal(t, "hello", fakeSvc.createPostReq.Content)
	assert.Equal(t, "public", fakeSvc.createPostReq.Visibility)
	assert.Equal(t, []string{"media-1"}, fakeSvc.createPostReq.MediaIDs)

	assert.Equal(t, "post-1", res.Id)
	assert.Equal(t, "author-1", res.AuthorId)
	assert.Equal(t, "hello", res.Content)
	assert.Equal(t, int32(1), res.LikeCount)
	require.NotNil(t, res.Author)
	assert.Equal(t, "author", res.Author.Username)
	require.Len(t, res.Media, 1)
	assert.Equal(t, "media-1", res.Media[0].Id)
	assert.Equal(t, int64(1024), res.Media[0].SizeBytes)
}

func TestCreatePostReturnsMappedError(t *testing.T) {
	fakeSvc := &fakeContentService{
		err: service.ErrInvalidPostContent,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.CreatePost(context.Background(), &contentpb.CreatePostRequest{})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetPostSuccess(t *testing.T) {
	fakeSvc := &fakeContentService{
		post: testPostResponse(),
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.GetPost(context.Background(), &contentpb.GetPostRequest{
		PostId:   "post-1",
		ViewerId: "viewer-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "post-1", fakeSvc.getPostReq.PostID)
	assert.Equal(t, "viewer-1", fakeSvc.getPostReq.ViewerID)
	assert.Equal(t, "post-1", res.Id)
}

func TestGetPostReturnsMappedNotFound(t *testing.T) {
	fakeSvc := &fakeContentService{
		err: repository.ErrPostNotFound,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.GetPost(context.Background(), &contentpb.GetPostRequest{
		PostId: "post-1",
	})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdatePostSuccess(t *testing.T) {
	fakeSvc := &fakeContentService{
		post: testPostResponse(),
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.UpdatePost(context.Background(), &contentpb.UpdatePostRequest{
		PostId:      "post-1",
		RequesterId: "user-1",
		Content:     "updated",
		Visibility:  "public",
		MediaIds:    []string{"media-1"},
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "post-1", fakeSvc.updatePostReq.PostID)
	assert.Equal(t, "user-1", fakeSvc.updatePostReq.RequesterID)
	assert.Equal(t, "updated", fakeSvc.updatePostReq.Content)
	assert.Equal(t, []string{"media-1"}, fakeSvc.updatePostReq.MediaIDs)
	assert.Equal(t, "post-1", res.Id)
}

func TestDeletePostSuccess(t *testing.T) {
	fakeSvc := &fakeContentService{}
	server := newTestContentServer(fakeSvc)

	res, err := server.DeletePost(context.Background(), &contentpb.DeletePostRequest{
		PostId:      "post-1",
		RequesterId: "user-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "post-1", fakeSvc.deletePostReq.PostID)
	assert.Equal(t, "user-1", fakeSvc.deletePostReq.RequesterID)
	assert.True(t, res.Success)
	assert.Equal(t, "post deleted successfully", res.Message)
}

func TestDeletePostReturnsMappedError(t *testing.T) {
	fakeSvc := &fakeContentService{
		deleteErr: repository.ErrForbidden,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.DeletePost(context.Background(), &contentpb.DeletePostRequest{
		PostId:      "post-1",
		RequesterId: "user-1",
	})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestCreateReplySuccess(t *testing.T) {
	post := testPostResponse()
	post.ReplyToPostID = "parent-post-1"

	fakeSvc := &fakeContentService{
		post: post,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.CreateReply(context.Background(), &contentpb.CreateReplyRequest{
		AuthorId:      "author-1",
		ReplyToPostId: "parent-post-1",
		Content:       "reply",
		Visibility:    "public",
		MediaIds:      []string{"media-1"},
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "author-1", fakeSvc.createReplyReq.AuthorID)
	assert.Equal(t, "parent-post-1", fakeSvc.createReplyReq.ReplyToPostID)
	assert.Equal(t, "reply", fakeSvc.createReplyReq.Content)
	assert.Equal(t, []string{"media-1"}, fakeSvc.createReplyReq.MediaIDs)
	assert.Equal(t, "parent-post-1", res.ReplyToPostId)
}

func TestGetRepliesSuccess(t *testing.T) {
	fakeSvc := &fakeContentService{
		posts: []dto.PostResponse{*testPostResponse()},
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.GetReplies(context.Background(), &contentpb.GetRepliesRequest{
		PostId: "post-1",
		Limit:  10,
		Cursor: "cursor-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Posts, 1)

	assert.Equal(t, "post-1", fakeSvc.getRepliesReq.PostID)
	assert.Equal(t, 10, fakeSvc.getRepliesReq.Limit)
	assert.Equal(t, "cursor-1", fakeSvc.getRepliesReq.Cursor)
	assert.Equal(t, "post-1", res.Posts[0].Id)
}

func TestLikePostSuccess(t *testing.T) {
	fakeSvc := &fakeContentService{
		action: testActionResponse("post liked successfully"),
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.LikePost(context.Background(), &contentpb.LikePostRequest{
		UserId: "user-1",
		PostId: "post-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "user-1", fakeSvc.likeReq.UserID)
	assert.Equal(t, "post-1", fakeSvc.likeReq.PostID)
	assert.True(t, res.Success)
	assert.Equal(t, "post liked successfully", res.Message)
}

func TestUnlikePostSuccess(t *testing.T) {
	fakeSvc := &fakeContentService{
		action: testActionResponse("post unliked successfully"),
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.UnlikePost(context.Background(), &contentpb.UnlikePostRequest{
		UserId: "user-1",
		PostId: "post-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "user-1", fakeSvc.unlikeReq.UserID)
	assert.Equal(t, "post-1", fakeSvc.unlikeReq.PostID)
	assert.True(t, res.Success)
	assert.Equal(t, "post unliked successfully", res.Message)
}

func TestBookmarkPostSuccess(t *testing.T) {
	fakeSvc := &fakeContentService{
		action: testActionResponse("post bookmarked successfully"),
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.BookmarkPost(context.Background(), &contentpb.BookmarkPostRequest{
		UserId: "user-1",
		PostId: "post-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "user-1", fakeSvc.bookmarkReq.UserID)
	assert.Equal(t, "post bookmarked successfully", res.Message)
}

func TestUnbookmarkPostSuccess(t *testing.T) {
	fakeSvc := &fakeContentService{
		action: testActionResponse("post unbookmarked successfully"),
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.UnbookmarkPost(context.Background(), &contentpb.UnbookmarkPostRequest{
		UserId: "user-1",
		PostId: "post-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "user-1", fakeSvc.unbookmarkReq.UserID)
	assert.Equal(t, "post unbookmarked successfully", res.Message)
}

func TestRepostPostSuccess(t *testing.T) {
	fakeSvc := &fakeContentService{
		action: testActionResponse("post reposted successfully"),
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.RepostPost(context.Background(), &contentpb.RepostPostRequest{
		UserId: "user-1",
		PostId: "post-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "user-1", fakeSvc.repostReq.UserID)
	assert.Equal(t, "post reposted successfully", res.Message)
}

func TestUndoRepostSuccess(t *testing.T) {
	fakeSvc := &fakeContentService{
		action: testActionResponse("repost removed successfully"),
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.UndoRepost(context.Background(), &contentpb.UndoRepostRequest{
		UserId: "user-1",
		PostId: "post-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "user-1", fakeSvc.undoRepostReq.UserID)
	assert.Equal(t, "repost removed successfully", res.Message)
}

func TestFollowUserSuccess(t *testing.T) {
	fakeSvc := &fakeContentService{
		action: testActionResponse("user followed successfully"),
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.FollowUser(context.Background(), &contentpb.FollowUserRequest{
		FollowerId:  "follower-1",
		FollowingId: "following-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "follower-1", fakeSvc.followReq.FollowerID)
	assert.Equal(t, "following-1", fakeSvc.followReq.FollowingID)
	assert.Equal(t, "user followed successfully", res.Message)
}

func TestUnfollowUserSuccess(t *testing.T) {
	fakeSvc := &fakeContentService{
		action: testActionResponse("user unfollowed successfully"),
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.UnfollowUser(context.Background(), &contentpb.UnfollowUserRequest{
		FollowerId:  "follower-1",
		FollowingId: "following-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "follower-1", fakeSvc.unfollowReq.FollowerID)
	assert.Equal(t, "following-1", fakeSvc.unfollowReq.FollowingID)
	assert.Equal(t, "user unfollowed successfully", res.Message)
}

func TestGetFollowersSuccess(t *testing.T) {
	fakeSvc := &fakeContentService{
		users: []dto.UserSummary{
			{
				ID:          "user-1",
				Username:    "follower",
				DisplayName: "Follower User",
				AvatarURL:   "https://example.com/avatar.png",
				IsVerified:  true,
			},
		},
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.GetFollowers(context.Background(), &contentpb.GetFollowersRequest{
		UserId: "user-1",
		Limit:  10,
		Cursor: "cursor-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Users, 1)

	assert.Equal(t, "user-1", fakeSvc.getFollowersReq.UserID)
	assert.Equal(t, 10, fakeSvc.getFollowersReq.Limit)
	assert.Equal(t, "cursor-1", fakeSvc.getFollowersReq.Cursor)

	assert.Equal(t, "user-1", res.Users[0].Id)
	assert.Equal(t, "follower", res.Users[0].Username)
	assert.True(t, res.Users[0].IsVerified)
}

func TestGetFollowingSuccess(t *testing.T) {
	fakeSvc := &fakeContentService{
		users: []dto.UserSummary{
			{
				ID:          "user-2",
				Username:    "following",
				DisplayName: "Following User",
			},
		},
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.GetFollowing(context.Background(), &contentpb.GetFollowingRequest{
		UserId: "user-1",
		Limit:  10,
		Cursor: "cursor-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Users, 1)

	assert.Equal(t, "user-1", fakeSvc.getFollowingReq.UserID)
	assert.Equal(t, "user-2", res.Users[0].Id)
	assert.Equal(t, "following", res.Users[0].Username)
}

func TestGetUserTimelineSuccess(t *testing.T) {
	post := *testPostResponse()

	fakeSvc := &fakeContentService{
		timeline: []dto.TimelineItemResponse{
			{
				Type:       "post",
				Post:       post,
				RepostedAt: "",
			},
			{
				Type:       "repost",
				Post:       post,
				RepostedAt: "2026-07-02T11:00:00Z",
			},
		},
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.GetUserTimeline(context.Background(), &contentpb.GetUserTimelineRequest{
		UserId: "user-1",
		Limit:  10,
		Cursor: "cursor-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Items, 2)

	assert.Equal(t, "user-1", fakeSvc.timelineReq.UserID)
	assert.Equal(t, 10, fakeSvc.timelineReq.Limit)
	assert.Equal(t, "cursor-1", fakeSvc.timelineReq.Cursor)

	assert.Equal(t, "post", res.Items[0].Type)
	assert.Equal(t, "post-1", res.Items[0].Post.Id)
	assert.Equal(t, "repost", res.Items[1].Type)
	assert.Equal(t, "2026-07-02T11:00:00Z", res.Items[1].RepostedAt)
}

func TestActionMethodReturnsMappedError(t *testing.T) {
	fakeSvc := &fakeContentService{
		err: repository.ErrUserNotFound,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.FollowUser(context.Background(), &contentpb.FollowUserRequest{
		FollowerId:  "missing-user",
		FollowingId: "user-2",
	})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestMapContentServiceError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{
			name: "invalid post content",
			err:  service.ErrInvalidPostContent,
			code: codes.InvalidArgument,
		},
		{
			name: "invalid post id",
			err:  service.ErrInvalidPostID,
			code: codes.InvalidArgument,
		},
		{
			name: "invalid author id",
			err:  service.ErrInvalidAuthorID,
			code: codes.InvalidArgument,
		},
		{
			name: "invalid user id",
			err:  service.ErrInvalidUserID,
			code: codes.InvalidArgument,
		},
		{
			name: "cannot follow self",
			err:  service.ErrCannotFollowSelf,
			code: codes.InvalidArgument,
		},
		{
			name: "post not found",
			err:  repository.ErrPostNotFound,
			code: codes.NotFound,
		},
		{
			name: "user not found",
			err:  repository.ErrUserNotFound,
			code: codes.NotFound,
		},
		{
			name: "forbidden",
			err:  repository.ErrForbidden,
			code: codes.PermissionDenied,
		},
		{
			name: "internal",
			err:  errors.New("database down"),
			code: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapContentServiceError(tt.err)

			require.Error(t, err)
			assert.Equal(t, tt.code, status.Code(err))
		})
	}
}

func TestCreateReplyReturnsMappedError(t *testing.T) {
	fakeSvc := &fakeContentService{
		err: service.ErrInvalidPostID,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.CreateReply(context.Background(), &contentpb.CreateReplyRequest{})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetRepliesReturnsMappedError(t *testing.T) {
	fakeSvc := &fakeContentService{
		err: repository.ErrPostNotFound,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.GetReplies(context.Background(), &contentpb.GetRepliesRequest{
		PostId: "post-1",
	})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestLikePostReturnsMappedError(t *testing.T) {
	fakeSvc := &fakeContentService{
		err: service.ErrInvalidUserID,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.LikePost(context.Background(), &contentpb.LikePostRequest{})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestUnlikePostReturnsMappedError(t *testing.T) {
	fakeSvc := &fakeContentService{
		err: repository.ErrPostNotFound,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.UnlikePost(context.Background(), &contentpb.UnlikePostRequest{})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestBookmarkPostReturnsMappedError(t *testing.T) {
	fakeSvc := &fakeContentService{
		err: service.ErrInvalidPostID,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.BookmarkPost(context.Background(), &contentpb.BookmarkPostRequest{})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestUnbookmarkPostReturnsMappedError(t *testing.T) {
	fakeSvc := &fakeContentService{
		err: service.ErrInvalidPostID,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.UnbookmarkPost(context.Background(), &contentpb.UnbookmarkPostRequest{})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestRepostPostReturnsMappedError(t *testing.T) {
	fakeSvc := &fakeContentService{
		err: repository.ErrForbidden,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.RepostPost(context.Background(), &contentpb.RepostPostRequest{})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestUndoRepostReturnsMappedError(t *testing.T) {
	fakeSvc := &fakeContentService{
		err: repository.ErrPostNotFound,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.UndoRepost(context.Background(), &contentpb.UndoRepostRequest{})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUnfollowUserReturnsMappedError(t *testing.T) {
	fakeSvc := &fakeContentService{
		err: service.ErrCannotFollowSelf,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.UnfollowUser(context.Background(), &contentpb.UnfollowUserRequest{})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetFollowersReturnsMappedError(t *testing.T) {
	fakeSvc := &fakeContentService{
		err: repository.ErrUserNotFound,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.GetFollowers(context.Background(), &contentpb.GetFollowersRequest{})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetFollowingReturnsMappedError(t *testing.T) {
	fakeSvc := &fakeContentService{
		err: repository.ErrUserNotFound,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.GetFollowing(context.Background(), &contentpb.GetFollowingRequest{})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetUserTimelineReturnsMappedError(t *testing.T) {
	fakeSvc := &fakeContentService{
		err: repository.ErrUserNotFound,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.GetUserTimeline(context.Background(), &contentpb.GetUserTimelineRequest{})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdatePostReturnsRawErrorCurrentBehavior(t *testing.T) {
	expectedErr := service.ErrInvalidPostContent

	fakeSvc := &fakeContentService{
		err: expectedErr,
	}
	server := newTestContentServer(fakeSvc)

	res, err := server.UpdatePost(context.Background(), &contentpb.UpdatePostRequest{})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}
