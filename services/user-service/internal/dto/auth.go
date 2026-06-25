package dto

type RegisterRequest struct {
	Email       string
	Username    string
	Password    string
	DisplayName string
	UserAgent   string
	IPAddress   string
}

type LoginRequest struct {
	EmailOrUsername string
	Password        string
	UserAgent       string
	IPAddress       string
}

type RefreshTokenRequest struct {
	RefreshToken string
	UserAgent    string
	IPAddress    string
}

type LogoutRequest struct {
	RefreshToken string
}

type AuthUserResponse struct {
	ID          string
	Email       string
	Username    string
	DisplayName string
	Role        string
	AvatarURL   string
	Message     string
}

type AuthResponse struct {
	User         AuthUserResponse
	AccessToken  string
	RefreshToken string
	Message      string
}
