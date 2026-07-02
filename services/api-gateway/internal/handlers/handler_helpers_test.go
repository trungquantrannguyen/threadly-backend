package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	contentpb "github.com/trungquantrannguyen/threadly/proto/content"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func setupHandlerTest() {
	gin.SetMode(gin.TestMode)
}

func newTestContext(method string, target string) (*gin.Context, *httptest.ResponseRecorder) {
	setupHandlerTest()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(method, target, nil)
	c.Request = req

	return c, w
}

func TestHealthHandlerCheck(t *testing.T) {
	setupHandlerTest()

	router := gin.New()
	handler := NewHealthHandler(config.Config{
		ServiceName: "api-gateway",
		AppEnv:      "test",
	})

	router.GET("/health", handler.Check)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "api-gateway")
	assert.Contains(t, w.Body.String(), "test")
}

func TestHandleGRPCErrorInvalidArgument(t *testing.T) {
	c, w := newTestContext(http.MethodGet, "/")

	HandleGRPCError(c, status.Error(codes.InvalidArgument, "invalid request"))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request")
}

func TestHandleGRPCErrorUnauthenticated(t *testing.T) {
	c, w := newTestContext(http.MethodGet, "/")

	HandleGRPCError(c, status.Error(codes.Unauthenticated, "unauthenticated"))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthenticated")
}

func TestHandleGRPCErrorPermissionDenied(t *testing.T) {
	c, w := newTestContext(http.MethodGet, "/")

	HandleGRPCError(c, status.Error(codes.PermissionDenied, "forbidden"))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "forbidden")
}

func TestHandleGRPCErrorNotFound(t *testing.T) {
	c, w := newTestContext(http.MethodGet, "/")

	HandleGRPCError(c, status.Error(codes.NotFound, "not found"))

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
}

func TestHandleGRPCErrorAlreadyExists(t *testing.T) {
	c, w := newTestContext(http.MethodGet, "/")

	HandleGRPCError(c, status.Error(codes.AlreadyExists, "already exists"))

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "already exists")
}

func TestHandleGRPCErrorDefaultInternal(t *testing.T) {
	c, w := newTestContext(http.MethodGet, "/")

	HandleGRPCError(c, status.Error(codes.Internal, "database down"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Internal server error")
}

func TestParseLimitDefault(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/")

	limit, ok := parseLimit(c)

	assert.True(t, ok)
	assert.Equal(t, int64(20), limit)
}

func TestParseLimitValid(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/?limit=25")

	limit, ok := parseLimit(c)

	assert.True(t, ok)
	assert.Equal(t, int64(25), limit)
}

func TestParseLimitInvalidString(t *testing.T) {
	c, w := newTestContext(http.MethodGet, "/?limit=abc")

	limit, ok := parseLimit(c)

	assert.False(t, ok)
	assert.Equal(t, int64(0), limit)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid limit")
}

func TestParseLimitZero(t *testing.T) {
	c, w := newTestContext(http.MethodGet, "/?limit=0")

	limit, ok := parseLimit(c)

	assert.False(t, ok)
	assert.Equal(t, int64(0), limit)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParseLimitTooLarge(t *testing.T) {
	c, w := newTestContext(http.MethodGet, "/?limit=51")

	limit, ok := parseLimit(c)

	assert.False(t, ok)
	assert.Equal(t, int64(0), limit)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestValidatePostActionSuccess(t *testing.T) {
	c, _ := newTestContext(http.MethodPost, "/")

	ok := validatePostAction(c, "post-1", "user-1")

	assert.True(t, ok)
}

func TestValidatePostActionMissingPostID(t *testing.T) {
	c, w := newTestContext(http.MethodPost, "/")

	ok := validatePostAction(c, "", "user-1")

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Missing postID")
}

func TestValidatePostActionMissingUserID(t *testing.T) {
	c, w := newTestContext(http.MethodPost, "/")

	ok := validatePostAction(c, "post-1", "")

	assert.False(t, ok)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestValidateUserActionSuccess(t *testing.T) {
	c, _ := newTestContext(http.MethodPost, "/")

	ok := validateUserAction(c, "requester-1", "target-1")

	assert.True(t, ok)
}

func TestValidateUserActionMissingRequesterID(t *testing.T) {
	c, w := newTestContext(http.MethodPost, "/")

	ok := validateUserAction(c, "", "target-1")

	assert.False(t, ok)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestValidateUserActionMissingTargetUserID(t *testing.T) {
	c, w := newTestContext(http.MethodPost, "/")

	ok := validateUserAction(c, "requester-1", "")

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Missing userID")
}

func TestToGatewayMediaResponses(t *testing.T) {
	res := toGatewayMediaResponses([]*contentpb.MediaResponse{
		{
			Id:        "media-1",
			Url:       "https://example.com/media.png",
			MimeType:  "image/png",
			SizeBytes: 1234,
			Width:     640,
			Height:    480,
		},
	})

	require.Len(t, res, 1)
	assert.Equal(t, "media-1", res[0].ID)
	assert.Equal(t, "https://example.com/media.png", res[0].URL)
	assert.Equal(t, "image/png", res[0].MimeType)
	assert.Equal(t, int64(1234), res[0].SizeBytes)
	assert.Equal(t, 640, res[0].Width)
	assert.Equal(t, 480, res[0].Height)
}

func TestToUserSummary(t *testing.T) {
	res := toUserSummary(&contentpb.UserSummary{
		Id:          "user-1",
		Username:    "trungquan",
		DisplayName: "Trung Quan",
		AvatarUrl:   "https://example.com/avatar.png",
		IsVerified:  true,
	})

	assert.Equal(t, "user-1", res.ID)
	assert.Equal(t, "trungquan", res.Username)
	assert.Equal(t, "Trung Quan", res.DisplayName)
	assert.Equal(t, "https://example.com/avatar.png", res.AvatarURL)
	assert.True(t, res.IsVerified)
}

func TestToPostResponse(t *testing.T) {
	res := toPostResponse(&contentpb.PostResponse{
		Id:            "post-1",
		AuthorId:      "author-1",
		ReplyToPostId: "parent-1",
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
				SizeBytes: 2048,
				Width:     300,
				Height:    200,
			},
		},
	})

	assert.Equal(t, "post-1", res.ID)
	assert.Equal(t, "author-1", res.AuthorID)
	assert.Equal(t, "parent-1", res.ReplyToPostID)
	assert.Equal(t, "hello", res.Content)
	assert.Equal(t, "public", res.Visibility)
	assert.Equal(t, 1, res.LikeCount)
	assert.Equal(t, 2, res.ReplyCount)
	assert.Equal(t, 3, res.RepostCount)
	assert.Equal(t, 4, res.BookmarkCount)
	assert.Equal(t, "author", res.Author.Username)
	require.Len(t, res.Medias, 1)
	assert.Equal(t, "media-1", res.Medias[0].ID)
}

func TestToGatewayPostResponse(t *testing.T) {
	res := toGatewayPostResponse(&contentpb.PostResponse{
		Id:            "post-1",
		AuthorId:      "author-1",
		ReplyToPostId: "",
		Content:       "hello from timeline",
		Visibility:    "public",
		CreatedAt:     "2026-07-02T10:00:00Z",
		UpdatedAt:     "2026-07-02T10:01:00Z",
		Author: &contentpb.UserSummary{
			Id:          "author-1",
			Username:    "author",
			DisplayName: "Author User",
		},
		Media: []*contentpb.MediaResponse{
			{
				Id:        "media-1",
				Url:       "https://example.com/media.png",
				MimeType:  "image/png",
				SizeBytes: 2048,
			},
		},
	})

	assert.Equal(t, "post-1", res.ID)
	assert.Equal(t, "author-1", res.AuthorID)
	assert.Equal(t, "hello from timeline", res.Content)
	assert.Equal(t, "author", res.Author.Username)
	require.Len(t, res.Medias, 1)
	assert.Equal(t, "media-1", res.Medias[0].ID)
}
