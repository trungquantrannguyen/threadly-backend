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
	}
}
