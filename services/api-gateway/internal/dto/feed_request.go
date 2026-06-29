package dto

type HomeFeedResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Data    HomeFeedData `json:"data"`
}

type HomeFeedData struct {
	Posts      []FeedPostResponse `json:"posts"`
	NextCursor string             `json:"next_cursor"`
}

type FeedPostResponse struct {
	ID            string          `json:"id"`
	AuthorID      string          `json:"author_id"`
	Content       string          `json:"content"`
	Visibility    string          `json:"visibility"`
	LikeCount     int             `json:"like_count"`
	ReplyCount    int             `json:"reply_count"`
	RepostCount   int             `json:"repost_count"`
	BookmarkCount int             `json:"bookmark_count"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     string          `json:"updated_at"`
	Author        FeedUserSummary `json:"author"`
}

type FeedUserSummary struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	IsVerified  bool   `json:"is_verified"`
}
