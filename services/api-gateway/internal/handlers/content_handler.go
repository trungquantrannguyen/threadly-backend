package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/middleware"
	"github.com/trungquantrannguyen/threadly/pkg/response"
	contentpb "github.com/trungquantrannguyen/threadly/proto/content"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/client"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/dto"
)

type ContentHandler struct {
	contentClient *client.ContentClient
	log           zerolog.Logger
}

func NewContentHandler(contentClient *client.ContentClient, log zerolog.Logger) *ContentHandler {
	return &ContentHandler{
		contentClient: contentClient,
		log:           log,
	}
}

// Register godoc
// @Summary Get Content service health
// @Description Get the status of content service
// @Tags Contents
// @Accept json
// @Produce json
// @Success 200 {object} dto.GetContentServiceHealthResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /contents/health [get]
func (h *ContentHandler) GetHealth(c *gin.Context) {
	health, err := h.contentClient.GetHealth(c.Request.Context())
	if err != nil {
		h.log.Error().
			Err(err).
			Msg("Failed to call content service health grpc method")

		response.Error(c, http.StatusServiceUnavailable, "Content service unavailable", err)
		return
	}

	response.OK(c, http.StatusOK, "Content service available", health)
}

// CreatePost godoc
// @Summary Create a post
// @Description Creates a new Threadly post of a user and returns the new post.
// @Tags Contents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreatePostRequest true "Create post request body"
// @Success 201 {object} dto.PostResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /contents/posts [post]
func (h *ContentHandler) CreatePost(c *gin.Context) {
	var req dto.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	authorID := middleware.GetUserID(c)
	if authorID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	res, err := h.contentClient.CreatePost(c.Request.Context(), &contentpb.CreatePostRequest{
		AuthorId:   authorID,
		Content:    req.Content,
		Visibility: req.Visibility,
		MediaIds:   req.MediaIDs,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to create post")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusCreated, "Created a post successfully", res)
}

// GetPost godoc
// @Summary Get a post
// @Description Returns the requested post.
// @Tags Contents
// @Produce json
// @Security BearerAuth
// @Param postID path string true "postID"
// @Success 200 {object} dto.PostResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /contents/posts/{postID} [get]
func (h *ContentHandler) GetPost(c *gin.Context) {
	postID := c.Param("postID")
	if postID == "" {
		response.Error(c, http.StatusBadRequest, "Missing params", nil)
		return
	}

	res, err := h.contentClient.GetPost(c.Request.Context(), &contentpb.GetPostRequest{
		PostId: postID,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get post")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusOK, "Get post successfully", toPostResponse(res))
}

// UpdatePost godoc
// @Summary Update a post
// @Description Updates the authenticated user's own post. Supports updating content, visibility, and attached media IDs.
// @Tags Contents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param postID path string true "Post ID"
// @Param request body dto.UpdatePostRequest true "Update post request body"
// @Success 200 {object} dto.PostResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /contents/posts/{postID} [patch]
func (h *ContentHandler) UpdatePost(c *gin.Context) {
	var req dto.UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	postID := c.Param("postID")
	if postID == "" {
		response.Error(c, http.StatusBadRequest, "Missing params", nil)
		return
	}

	requesterID := middleware.GetUserID(c)
	if requesterID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	res, err := h.contentClient.UpdatePost(c.Request.Context(), &contentpb.UpdatePostRequest{
		PostId:      postID,
		RequesterId: requesterID,
		Content:     req.Content,
		Visibility:  req.Visibility,
		MediaIds:    req.MediaIDs,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to update post")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusOK, "Updated post successfully", res)
}

// DeletePost godoc
// @Summary Delete a post
// @Description Returns the delete post status.
// @Tags Contents
// @Produce json
// @Security BearerAuth
// @Param postID path string true "postID"
// @Success 204 {object} interface{}
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /contents/posts/{postID} [delete]
func (h *ContentHandler) DeletePost(c *gin.Context) {
	postID := c.Param("postID")
	if postID == "" {
		response.Error(c, http.StatusBadRequest, "Missing params", nil)
		return
	}

	requestID := middleware.GetUserID(c)

	if requestID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	res, err := h.contentClient.DeletePost(c.Request.Context(), &contentpb.DeletePostRequest{
		PostId:      postID,
		RequesterId: requestID,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to delete post")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusNoContent, "Delete post successfully", res)
}

// CreateReply godoc
// @Summary Create a reply post
// @Description Creates a new reply to a post and returns the new reply.
// @Tags Contents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param postID path string true "postID"
// @Param request body dto.CreatePostRequest true "Create reply request body"
// @Success 201 {object} dto.PostResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /contents/posts/{postID}/replies [post]
func (h *ContentHandler) CreateReply(c *gin.Context) {
	var req dto.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	postID := c.Param("postID")
	if postID == "" {
		response.Error(c, http.StatusBadRequest, "Missing params", nil)
		return
	}

	authorID := middleware.GetUserID(c)
	if authorID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	res, err := h.contentClient.CreateReply(c.Request.Context(), &contentpb.CreateReplyRequest{
		AuthorId:      authorID,
		Content:       req.Content,
		ReplyToPostId: postID,
		MediaIds:      req.MediaIDs,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to create reply")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusCreated, "Create reply successfully", res)
}

// GetReplies godoc
// @Summary Get replies of a post
// @Description Returns replies for a post.
// @Tags Contents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param postID path string true "postID"
// @Param request query dto.GetRepliesRequest true "Get replies request body"
// @Success 200 {object} []dto.PostResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /contents/posts/{postID}/replies [get]
func (h *ContentHandler) GetReplies(c *gin.Context) {
	limit := int64(20)
	limitQuery := c.Query("limit")
	if limitQuery != "" {
		parsedLimit, err := strconv.ParseInt(limitQuery, 10, 32)
		if err != nil || parsedLimit < 0 || parsedLimit > 50 {
			response.Error(c, http.StatusBadRequest, "Invalid limit", err)
			return
		}
		limit = parsedLimit
	}
	cursor := c.Query("cursor")
	postID := c.Param("postID")
	if postID == "" {
		response.Error(c, http.StatusBadRequest, "Missing params", nil)
		return
	}

	res, err := h.contentClient.GetReplies(c.Request.Context(), &contentpb.GetRepliesRequest{
		PostId: postID,
		Limit:  int32(limit),
		Cursor: cursor,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get post")
		HandleGRPCError(c, err)
		return
	}

	posts := make([]dto.PostResponse, 0, len(res.GetPosts()))

	for _, post := range res.GetPosts() {
		posts = append(posts, toPostResponse(post))
	}

	response.OK(c, http.StatusOK, "Get replies successfully", posts)
}

// LikePost godoc
// @Summary Like a post
// @Description Likes a post as the authenticated user.
// @Tags Contents
// @Produce json
// @Security BearerAuth
// @Param postID path string true "postID"
// @Success 200 {object} interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /contents/posts/{postID}/likes [post]
func (h *ContentHandler) LikePost(c *gin.Context) {
	postID := c.Param("postID")
	if postID == "" {
		response.Error(c, http.StatusBadRequest, "Missing postID", nil)
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	res, err := h.contentClient.LikePost(c.Request.Context(), &contentpb.LikePostRequest{
		UserId: userID,
		PostId: postID,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to like post")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusOK, res.GetMessage(), res)
}

// UnlikePost godoc
// @Summary Unlike a post
// @Description Removes the authenticated user's like from a post.
// @Tags Contents
// @Produce json
// @Security BearerAuth
// @Param postID path string true "postID"
// @Success 200 {object} interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /contents/posts/{postID}/likes [delete]
func (h *ContentHandler) UnlikePost(c *gin.Context) {
	postID := c.Param("postID")
	if postID == "" {
		response.Error(c, http.StatusBadRequest, "Missing postID", nil)
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	res, err := h.contentClient.UnlikePost(c.Request.Context(), &contentpb.UnlikePostRequest{
		UserId: userID,
		PostId: postID,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to unlike post")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusOK, res.GetMessage(), res)
}

// BookmarkPost godoc
// @Summary Bookmark a post
// @Description Bookmarks a post as the authenticated user.
// @Tags Contents
// @Produce json
// @Security BearerAuth
// @Param postID path string true "postID"
// @Success 200 {object} interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /contents/posts/{postID}/bookmarks [post]
func (h *ContentHandler) BookmarkPost(c *gin.Context) {
	postID := c.Param("postID")
	userID := middleware.GetUserID(c)

	if !validatePostAction(c, postID, userID) {
		return
	}

	res, err := h.contentClient.BookmarkPost(c.Request.Context(), &contentpb.BookmarkPostRequest{
		UserId: userID,
		PostId: postID,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to bookmark post")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusOK, res.GetMessage(), res)
}

// UnbookmarkPost godoc
// @Summary Unbookmark a post
// @Description Removes the authenticated user's bookmark from a post.
// @Tags Contents
// @Produce json
// @Security BearerAuth
// @Param postID path string true "postID"
// @Success 200 {object} interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /contents/posts/{postID}/bookmarks [delete]
func (h *ContentHandler) UnbookmarkPost(c *gin.Context) {
	postID := c.Param("postID")
	userID := middleware.GetUserID(c)

	if !validatePostAction(c, postID, userID) {
		return
	}

	res, err := h.contentClient.UnbookmarkPost(c.Request.Context(), &contentpb.UnbookmarkPostRequest{
		UserId: userID,
		PostId: postID,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to unbookmark post")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusOK, res.GetMessage(), res)
}

// RepostPost godoc
// @Summary Repost a post
// @Description Reposts a post as the authenticated user.
// @Tags Contents
// @Produce json
// @Security BearerAuth
// @Param postID path string true "postID"
// @Success 200 {object} interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /contents/posts/{postID}/reposts [post]
func (h *ContentHandler) RepostPost(c *gin.Context) {
	postID := c.Param("postID")
	userID := middleware.GetUserID(c)

	if !validatePostAction(c, postID, userID) {
		return
	}

	res, err := h.contentClient.RepostPost(c.Request.Context(), &contentpb.RepostPostRequest{
		UserId: userID,
		PostId: postID,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to repost post")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusOK, res.GetMessage(), res)
}

// UndoRepost godoc
// @Summary Undo repost
// @Description Removes the authenticated user's repost from a post.
// @Tags Contents
// @Produce json
// @Security BearerAuth
// @Param postID path string true "postID"
// @Success 200 {object} interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /contents/posts/{postID}/reposts [delete]
func (h *ContentHandler) UndoRepost(c *gin.Context) {
	postID := c.Param("postID")
	userID := middleware.GetUserID(c)

	if !validatePostAction(c, postID, userID) {
		return
	}

	res, err := h.contentClient.UndoRepost(c.Request.Context(), &contentpb.UndoRepostRequest{
		UserId: userID,
		PostId: postID,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to undo repost")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusOK, res.GetMessage(), res)
}

// FollowUser godoc
// @Summary Follow a user
// @Description Authenticated user follows another user.
// @Tags Contents
// @Produce json
// @Security BearerAuth
// @Param userID path string true "target userID"
// @Success 200 {object} interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /contents/users/{userID}/follow [post]
func (h *ContentHandler) FollowUser(c *gin.Context) {
	followerID := middleware.GetUserID(c)
	followingID := c.Param("userID")

	if !validateUserAction(c, followerID, followingID) {
		return
	}

	res, err := h.contentClient.FollowUser(c.Request.Context(), &contentpb.FollowUserRequest{
		FollowerId:  followerID,
		FollowingId: followingID,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to follow user")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusOK, res.GetMessage(), res)
}

// UnfollowUser godoc
// @Summary Unfollow a user
// @Description Authenticated user unfollows another user.
// @Tags Contents
// @Produce json
// @Security BearerAuth
// @Param userID path string true "target userID"
// @Success 200 {object} interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /contents/users/{userID}/follow [delete]
func (h *ContentHandler) UnfollowUser(c *gin.Context) {
	followerID := middleware.GetUserID(c)
	followingID := c.Param("userID")

	if !validateUserAction(c, followerID, followingID) {
		return
	}

	res, err := h.contentClient.UnfollowUser(c.Request.Context(), &contentpb.UnfollowUserRequest{
		FollowerId:  followerID,
		FollowingId: followingID,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to unfollow user")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusOK, res.GetMessage(), res)
}

// GetFollowers godoc
// @Summary Get user followers
// @Description Returns users who follow the requested user.
// @Tags Contents
// @Produce json
// @Security BearerAuth
// @Param userID path string true "userID"
// @Param limit query int false "limit"
// @Param cursor query string false "cursor"
// @Success 200 {object} []dto.UserSummary
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /contents/users/{userID}/followers [get]
func (h *ContentHandler) GetFollowers(c *gin.Context) {
	limit, ok := parseLimit(c)
	if !ok {
		return
	}

	userID := c.Param("userID")
	if userID == "" {
		response.Error(c, http.StatusBadRequest, "Missing userID", nil)
		return
	}

	res, err := h.contentClient.GetFollowers(c.Request.Context(), &contentpb.GetFollowersRequest{
		UserId: userID,
		Limit:  int32(limit),
		Cursor: c.Query("cursor"),
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get followers")
		HandleGRPCError(c, err)
		return
	}

	users := make([]dto.UserSummary, 0, len(res.GetUsers()))
	for _, user := range res.GetUsers() {
		users = append(users, toUserSummary(user))
	}

	response.OK(c, http.StatusOK, "Get followers successfully", users)
}

// GetFollowing godoc
// @Summary Get users followed by a user
// @Description Returns users that the requested user follows.
// @Tags Contents
// @Produce json
// @Security BearerAuth
// @Param userID path string true "userID"
// @Param limit query int false "limit"
// @Param cursor query string false "cursor"
// @Success 200 {object} []dto.UserSummary
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /contents/users/{userID}/following [get]
func (h *ContentHandler) GetFollowing(c *gin.Context) {
	limit, ok := parseLimit(c)
	if !ok {
		return
	}

	userID := c.Param("userID")
	if userID == "" {
		response.Error(c, http.StatusBadRequest, "Missing userID", nil)
		return
	}

	res, err := h.contentClient.GetFollowing(c.Request.Context(), &contentpb.GetFollowingRequest{
		UserId: userID,
		Limit:  int32(limit),
		Cursor: c.Query("cursor"),
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get following")
		HandleGRPCError(c, err)
		return
	}

	users := make([]dto.UserSummary, 0, len(res.GetUsers()))
	for _, user := range res.GetUsers() {
		users = append(users, toUserSummary(user))
	}

	response.OK(c, http.StatusOK, "Get following successfully", users)
}

func parseLimit(c *gin.Context) (int64, bool) {
	limit := int64(20)
	limitQuery := c.Query("limit")

	if limitQuery == "" {
		return limit, true
	}

	parsedLimit, err := strconv.ParseInt(limitQuery, 10, 32)
	if err != nil || parsedLimit <= 0 || parsedLimit > 50 {
		response.Error(c, http.StatusBadRequest, "Invalid limit", err)
		return 0, false
	}

	return parsedLimit, true
}

func validatePostAction(c *gin.Context, postID string, userID string) bool {
	if postID == "" {
		response.Error(c, http.StatusBadRequest, "Missing postID", nil)
		return false
	}

	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return false
	}

	return true
}

func validateUserAction(c *gin.Context, requesterID string, targetUserID string) bool {
	if requesterID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return false
	}

	if targetUserID == "" {
		response.Error(c, http.StatusBadRequest, "Missing userID", nil)
		return false
	}

	return true
}

func toPostResponse(post *contentpb.PostResponse) dto.PostResponse {
	author := post.GetAuthor()

	return dto.PostResponse{
		ID:            post.GetId(),
		AuthorID:      post.GetAuthorId(),
		ReplyToPostID: post.GetReplyToPostId(),
		Content:       post.GetContent(),
		Visibility:    post.GetVisibility(),

		LikeCount:     int(post.GetLikeCount()),
		ReplyCount:    int(post.GetReplyCount()),
		RepostCount:   int(post.GetRepostCount()),
		BookmarkCount: int(post.GetBookmarkCount()),

		CreatedAt: post.GetCreatedAt(),
		UpdatedAt: post.GetUpdatedAt(),

		Author: dto.UserSummary{
			ID:          author.GetId(),
			Username:    author.GetUsername(),
			DisplayName: author.GetDisplayName(),
			AvatarURL:   author.GetAvatarUrl(),
			IsVerified:  author.GetIsVerified(),
		},
		Medias: toGatewayMediaResponses(post.Media),
	}
}

func toUserSummary(user *contentpb.UserSummary) dto.UserSummary {
	return dto.UserSummary{
		ID:          user.GetId(),
		Username:    user.GetUsername(),
		DisplayName: user.GetDisplayName(),
		AvatarURL:   user.GetAvatarUrl(),
		IsVerified:  user.GetIsVerified(),
	}
}

func toGatewayMediaResponses(media []*contentpb.MediaResponse) []dto.MediaResponse {
	res := make([]dto.MediaResponse, 0, len(media))

	for _, item := range media {
		res = append(res, dto.MediaResponse{
			ID:        item.GetId(),
			URL:       item.GetUrl(),
			MimeType:  item.GetMimeType(),
			SizeBytes: item.GetSizeBytes(),
			Width:     int(item.GetWidth()),
			Height:    int(item.GetHeight()),
		})
	}

	return res
}
