package dto

type ResgisterReponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Data    AuthData `json:"data"`
}

type LoginResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Data    AuthData `json:"data"`
}

type RefreshTokenResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Data    AuthData `json:"data"`
}

type LogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type GetUserServiceHealthResponse struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message"`
	Data    GetUserServiceHealthData `json:"data"`
}

type GetUserServiceHealthData struct {
	Service   string `json:"service"`
	Status    string `json:"status"`
	Env       string `json:"env"`
	CheckedAt string `json:"checked_at"`
}

type GetMeResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Data    UserData `json:"data"`
}

type AuthData struct {
	User         UserData `json:"user"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
}

type UserData struct {
	Id          string `json:"id"`
	Email       string `json:"email"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	AvatarURL   string `json:"avatar_url"`
}
