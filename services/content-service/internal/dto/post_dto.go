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

type LikePostRequest struct {
	UserID string
	PostID string
}

type UnlikePostRequest struct {
	UserID string
	PostID string
}

type BookmarkPostRequest struct {
	UserID string
	PostID string
}

type UnbookmarkPostRequest struct {
	UserID string
	PostID string
}

type RepostPostRequest struct {
	UserID string
	PostID string
}

type UndoRepostRequest struct {
	UserID string
	PostID string
}

type FollowUserRequest struct {
	FollowerID  string
	FollowingID string
}

type UnfollowUserRequest struct {
	FollowerID  string
	FollowingID string
}

type GetFollowersRequest struct {
	UserID string
	Limit  int
	Cursor string
}

type GetFollowingRequest struct {
	UserID string
	Limit  int
	Cursor string
}

type ActionResponse struct {
	Success bool
	Message string
}

type GetUserTimelineRequest struct {
	UserID string
	Limit  int
	Cursor string
}

type TimelineItemResponse struct {
	Type       string
	Post       PostResponse
	RepostedAt string
}
