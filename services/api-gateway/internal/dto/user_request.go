package dto

type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name" binding:"required,min=1,max=100"`
}

type LoginRequest struct {
	EmailOrUsername string `json:"email_or_username" binding:"required"`
	Password        string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type UpdateProfileRequest struct {
	DisplayName *string `json:"display_name" binding:"omitempty,min=1,max=80"`
	Bio         *string `json:"bio" binding:"omitempty,max=280"`
	AvatarURL   *string `json:"avatar_url" binding:"omitempty,max=2048"`
	BannerURL   *string `json:"banner_url" binding:"omitempty,max=2048"`
	Location    *string `json:"location" binding:"omitempty,max=100"`
	WebsiteURL  *string `json:"website_url" binding:"omitempty,max=2048"`
}
