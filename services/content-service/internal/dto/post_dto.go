package dto

type CreatePostRequest struct {
	AuthorID   string
	Content    string
	Visibility string
}

type GetPostRequest struct {
	PostID   string
	ViewerID string
}

type DeletePostRequest struct {
	PostID      string
	RequesterID string
}

type CreateReplyRequest struct {
	AuthorID      string
	ReplyToPostID string
	Content       string
	Visibility    string
}

type GetRepliesRequest struct {
	PostID string
	Limit  int
	Cursor string
}

type PostResponse struct {
	ID            string
	AuthorID      string
	ReplyToPostID string
	Content       string
	Visibility    string
	LikeCount     int
	ReplyCount    int
	RepostCount   int
	BookmarkCount int
	CreatedAt     string
	UpdatedAt     string
	Author        UserSummary
}

type UserSummary struct {
	ID          string
	Username    string
	DisplayName string
	AvatarURL   string
	IsVerified  bool
}
