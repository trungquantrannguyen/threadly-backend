package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/middleware"
	"github.com/trungquantrannguyen/threadly/pkg/response"
	contentpb "github.com/trungquantrannguyen/threadly/proto/content"
	userpb "github.com/trungquantrannguyen/threadly/proto/user"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/client"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/dto"
)

type GetMeWithPostsResponse struct {
	User  *userpb.AuthUserResponse `json:"user"`
	Posts []TimelineItemResponse   `json:"posts"`
}

type TimelineItemResponse struct {
	Type       string           `json:"type"`
	Post       dto.PostResponse `json:"post"`
	RepostedAt string           `json:"reposted_at,omitempty"`
}

type UserHandler struct {
	userClient    *client.UserClient
	contentClient *client.ContentClient
	log           zerolog.Logger
}

func NewUserHandler(userClient *client.UserClient, log zerolog.Logger, contentClient *client.ContentClient) *UserHandler {
	return &UserHandler{
		userClient:    userClient,
		contentClient: contentClient,
		log:           log,
	}
}

// Register godoc
// @Summary Get User service health
// @Description Get the status of user service
// @Tags Users
// @Accept json
// @Produce json
// @Success 200 {object} dto.GetUserServiceHealthResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/health [get]
func (h *UserHandler) GetHealth(c *gin.Context) {
	health, err := h.userClient.GetHealth(c.Request.Context())
	if err != nil {
		h.log.Error().
			Err(err).
			Msg("failed to call user service health grpc method")

		response.Error(c, http.StatusServiceUnavailable, "user service unavailable", err)
		return
	}

	response.OK(c, http.StatusOK, "user service available", health)
}

// Register godoc
// @Summary Register a new user
// @Description Creates a new Threadly user account and returns access and refresh tokens.
// @Tags Users
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Register request body"
// @Success 201 {object} dto.RegisterResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	res, err := h.userClient.Register(c.Request.Context(), &userpb.RegisterRequest{
		Email:       req.Email,
		Username:    req.Username,
		Password:    req.Password,
		DisplayName: req.DisplayName,
		UserAgent:   c.Request.UserAgent(),
		IpAddress:   c.ClientIP(),
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to register user")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusCreated, "Register successfully", res)
}

// Login godoc
// @Summary Login user
// @Description Authenticates a user using email/username and password.
// @Tags Users
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login request body"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	res, err := h.userClient.Login(c.Request.Context(), &userpb.LoginRequest{
		EmailOrUsername: req.EmailOrUsername,
		Password:        req.Password,
		UserAgent:       c.Request.UserAgent(),
		IpAddress:       c.ClientIP(),
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to login user")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusOK, "Login successfully", res)
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Uses a valid refresh token to issue a new access token.
// @Tags Users
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "Refresh token request body"
// @Success 200 {object} dto.RefreshTokenResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/refresh [post]
func (h *UserHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	res, err := h.userClient.RefreshToken(c.Request.Context(), &userpb.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
		UserAgent:    c.Request.UserAgent(),
		IpAddress:    c.ClientIP(),
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to refresh token")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusOK, "Token refreshed successfully", res)
}

// Logout godoc
// @Summary Logout user
// @Description Revokes the current refresh token session.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.LogoutRequest true "Logout request body"
// @Success 200 {object} dto.LogoutResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/logout [post]
func (h *UserHandler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	_, err := h.userClient.Logout(c.Request.Context(), &userpb.LogoutRequest{
		RefreshToken: req.RefreshToken,
		UserID:       userID,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to log out user")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusOK, "Logged out successfully", nil)
}

// GetMe godoc
// @Summary Get current user
// @Description Returns the authenticated user's profile.
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.GetMeResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	userRes, err := h.userClient.GetMe(c.Request.Context(), &userpb.GetMeRequest{
		UserID: userID,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get user")
		HandleGRPCError(c, err)
		return
	}

	timelineRes, err := h.contentClient.GetUserTimeline(c.Request.Context(), &contentpb.GetUserTimelineRequest{
		UserId: userID,
		Limit:  20,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get user timeline")
		HandleGRPCError(c, err)
		return
	}

	posts := make([]TimelineItemResponse, 0, len(timelineRes.GetItems()))

	for _, item := range timelineRes.GetItems() {
		posts = append(posts, TimelineItemResponse{
			Type:       item.GetType(),
			Post:       toGatewayPostResponse(item.GetPost()),
			RepostedAt: item.GetRepostedAt(),
		})
	}

	response.OK(c, http.StatusOK, "Get user successfully", GetMeWithPostsResponse{
		User:  userRes,
		Posts: posts,
	})
}

// UpdateProfile godoc
// @Summary Update current user profile
// @Description Updates the authenticated user's profile fields.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.UpdateProfileRequest true "Update profile request body"
// @Success 200 {object} dto.UpdateProfileResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/me [patch]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	res, err := h.userClient.UpdateProfile(c.Request.Context(), &userpb.UpdateProfileRequest{
		UserID:      userID,
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
		AvatarURL:   req.AvatarURL,
		BannerURL:   req.BannerURL,
		Location:    req.Location,
		WebsiteURL:  req.WebsiteURL,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to update profile")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusOK, "Profile updated successfully", res)
}

// DeleteUser godoc
// @Summary Delete current user
// @Description Soft deletes the authenticated user's account.
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.DeleteUserResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/me [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	_, err := h.userClient.DeleteUser(c.Request.Context(), &userpb.DeleteUserRequest{
		UserID: userID,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to delete user")
		HandleGRPCError(c, err)
		return
	}

	response.OK(c, http.StatusOK, "User deleted successfully", nil)
}

func toGatewayPostResponse(post *contentpb.PostResponse) dto.PostResponse {
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
		Medias: toGatewayMediaResponses(post.GetMedia()),
	}
}
