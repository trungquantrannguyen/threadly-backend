package dto

type CreatePostRequest struct {
	Content    string   `json:"content" binding:"required"`
	Visibility string   `json:"visibility"`
	MediaIDs   []string `json:"media_ids"`
}

type GetPostRequest struct {
	ViewerID string `json:"viewer_id"`
}

type GetRepliesRequest struct {
	Limit  int    `json:"limit" binding:"required,min=0,max=50"`
	Cursor string `json:"cursor"`
}

type UpdatePostRequest struct {
	Content    string   `json:"content" binding:"required"`
	Visibility string   `json:"visibility"`
	MediaIDs   []string `json:"media_ids"`
}

type PostResponse struct {
	ID            string          `json:"id"`
	AuthorID      string          `json:"author_id"`
	ReplyToPostID string          `json:"reply_to_post_id"`
	Content       string          `json:"content"`
	Visibility    string          `json:"visibility"`
	LikeCount     int             `json:"like_count"`
	ReplyCount    int             `json:"reply_count"`
	RepostCount   int             `json:"repost_count"`
	BookmarkCount int             `json:"bookmark_count"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     string          `json:"updated_at"`
	Author        UserSummary     `json:"author"`
	Medias        []MediaResponse `json:"medias"`
}

type GetContentServiceHealthResponse struct {
	Success bool                        `json:"success"`
	Message string                      `json:"message"`
	Data    GetContentServiceHealthData `json:"data"`
}

type GetContentServiceHealthData struct {
	Service   string `json:"service"`
	Status    string `json:"status"`
	Env       string `json:"env"`
	CheckedAt string `json:"checked_at"`
}

type GetFeedServiceHealthResponse struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message"`
	Data    GetFeedServiceHealthData `json:"data"`
}

type GetFeedServiceHealthData struct {
	Service   string `json:"service"`
	Status    string `json:"status"`
	Env       string `json:"env"`
	CheckedAt string `json:"checked_at"`
}

type GetNotificationServiceHealthResponse struct {
	Success bool                             `json:"success"`
	Message string                           `json:"message"`
	Data    GetNotificationServiceHealthData `json:"data"`
}

type GetNotificationServiceHealthData struct {
	Service   string `json:"service"`
	Status    string `json:"status"`
	Env       string `json:"env"`
	CheckedAt string `json:"checked_at"`
}

type GetStorageServiceHealthResponse struct {
	Success bool                        `json:"success"`
	Message string                      `json:"message"`
	Data    GetStorageServiceHealthData `json:"data"`
}

type GetStorageServiceHealthData struct {
	Service   string `json:"service"`
	Status    string `json:"status"`
	Env       string `json:"env"`
	CheckedAt string `json:"checked_at"`
}

type UserSummary struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	IsVerified  bool   `json:"is_verified"`
}

type MediaResponse struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}
