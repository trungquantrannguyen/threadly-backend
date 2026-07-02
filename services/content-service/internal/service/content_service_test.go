package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"github.com/trungquantrannguyen/threadly/pkg/messaging"
	"github.com/trungquantrannguyen/threadly/services/content-service/internal/dto"
	"github.com/trungquantrannguyen/threadly/services/content-service/internal/repository"
)

type publishedContentEvent struct {
	routingKey string
	event      messaging.Event
}

type fakeContentPublisher struct {
	published []publishedContentEvent
	err       error
}

func (f *fakeContentPublisher) Publish(ctx context.Context, routingKey string, payload any) error {
	event, _ := payload.(messaging.Event)

	f.published = append(f.published, publishedContentEvent{
		routingKey: routingKey,
		event:      event,
	})

	if f.err != nil {
		return f.err
	}

	return nil
}

type fakePostRepo struct {
	posts   map[string]*dbmodel.Post
	replies map[string][]dbmodel.Post

	createErr          error
	findErr            error
	updateErr          error
	deleteErr          error
	repliesErr         error
	incrementErr       error
	attachErr          error
	replaceErr         error
	timelineErr        error
	findTimelineResult []repository.UserTimelineItem

	createdPost *dbmodel.Post

	attachedPostID    uuid.UUID
	attachedUploader  uuid.UUID
	attachedMediaIDs  []string
	replacedPostID    uuid.UUID
	replacedUploader  uuid.UUID
	replacedMediaIDs  []string
	updatedPostID     string
	updatedRequester  string
	updatedContent    string
	updatedVisibility string
	deletedPostID     string
	deletedRequester  string
	incrementPostID   string
}

func newFakePostRepo() *fakePostRepo {
	return &fakePostRepo{
		posts:   map[string]*dbmodel.Post{},
		replies: map[string][]dbmodel.Post{},
	}
}

func testUser(id uuid.UUID, username string) dbmodel.User {
	now := time.Now().UTC()

	return dbmodel.User{
		ID:          id,
		Email:       username + "@example.com",
		Username:    username,
		DisplayName: "Test User",
		Role:        "user",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func testPost(id uuid.UUID, authorID uuid.UUID, content string) *dbmodel.Post {
	now := time.Now().UTC()

	return &dbmodel.Post{
		ID:         id,
		AuthorID:   authorID,
		Content:    content,
		Visibility: "public",
		CreatedAt:  now,
		UpdatedAt:  now,
		Author:     testUser(authorID, "author"),
	}
}

func normalizePost(post *dbmodel.Post) {
	now := time.Now().UTC()

	if post.ID == uuid.Nil {
		post.ID = uuid.New()
	}

	if post.CreatedAt.IsZero() {
		post.CreatedAt = now
	}

	if post.UpdatedAt.IsZero() {
		post.UpdatedAt = now
	}

	if post.Visibility == "" {
		post.Visibility = "public"
	}

	if post.Author.ID == uuid.Nil {
		post.Author = testUser(post.AuthorID, "author")
	}
}

func copyPost(post *dbmodel.Post) *dbmodel.Post {
	copied := *post
	return &copied
}

func (f *fakePostRepo) seedPost(post *dbmodel.Post) {
	normalizePost(post)
	f.posts[post.ID.String()] = copyPost(post)
}

func (f *fakePostRepo) Create(ctx context.Context, post *dbmodel.Post) error {
	if f.createErr != nil {
		return f.createErr
	}

	normalizePost(post)

	copied := copyPost(post)
	f.createdPost = copied
	f.posts[post.ID.String()] = copied

	return nil
}

func (f *fakePostRepo) FindByID(ctx context.Context, postID string) (*dbmodel.Post, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}

	post, exists := f.posts[postID]
	if !exists {
		return nil, repository.ErrPostNotFound
	}

	return copyPost(post), nil
}

func (f *fakePostRepo) UpdateOwnPost(ctx context.Context, postID string, requesterID string, content string, visibility string) (*dbmodel.Post, error) {
	f.updatedPostID = postID
	f.updatedRequester = requesterID
	f.updatedContent = content
	f.updatedVisibility = visibility

	if f.updateErr != nil {
		return nil, f.updateErr
	}

	post, exists := f.posts[postID]
	if !exists || post.AuthorID.String() != requesterID {
		return nil, repository.ErrPostNotFound
	}

	post.Content = content
	post.Visibility = visibility
	post.UpdatedAt = time.Now().UTC()

	return copyPost(post), nil
}

func (f *fakePostRepo) DeleteOwnPost(ctx context.Context, postID string, requesterID string) error {
	f.deletedPostID = postID
	f.deletedRequester = requesterID

	if f.deleteErr != nil {
		return f.deleteErr
	}

	post, exists := f.posts[postID]
	if !exists || post.AuthorID.String() != requesterID {
		return repository.ErrPostNotFound
	}

	delete(f.posts, postID)
	return nil
}

func (f *fakePostRepo) FindReplies(ctx context.Context, postID string, limit int) ([]dbmodel.Post, error) {
	if f.repliesErr != nil {
		return nil, f.repliesErr
	}

	return f.replies[postID], nil
}

func (f *fakePostRepo) IncrementReplyCount(ctx context.Context, postID string) error {
	f.incrementPostID = postID

	if f.incrementErr != nil {
		return f.incrementErr
	}

	post, exists := f.posts[postID]
	if exists {
		post.ReplyCount++
	}

	return nil
}

func (f *fakePostRepo) FindUserTimeline(ctx context.Context, userID uuid.UUID, limit int) ([]repository.UserTimelineItem, error) {
	if f.timelineErr != nil {
		return nil, f.timelineErr
	}

	return f.findTimelineResult, nil
}

func (f *fakePostRepo) AttachMediaToPost(ctx context.Context, postID uuid.UUID, uploaderID uuid.UUID, mediaIDs []string) error {
	f.attachedPostID = postID
	f.attachedUploader = uploaderID
	f.attachedMediaIDs = mediaIDs

	if f.attachErr != nil {
		return f.attachErr
	}

	return nil
}

func (f *fakePostRepo) ReplacePostMedia(ctx context.Context, postID uuid.UUID, uploaderID uuid.UUID, mediaIDs []string) error {
	f.replacedPostID = postID
	f.replacedUploader = uploaderID
	f.replacedMediaIDs = mediaIDs

	if f.replaceErr != nil {
		return f.replaceErr
	}

	return nil
}

type fakeInteractionRepo struct {
	userExistsResult bool
	userExistsErr    error
	existingUsers    map[uuid.UUID]bool

	likeCreated bool
	likeErr     error

	unlikeDeleted bool
	unlikeErr     error

	bookmarkCreated bool
	bookmarkErr     error

	unbookmarkDeleted bool
	unbookmarkErr     error

	repostCreated bool
	repostErr     error

	undoRepostDeleted bool
	undoRepostErr     error

	followCreated bool
	followErr     error

	unfollowDeleted bool
	unfollowErr     error

	followers    []dbmodel.User
	followersErr error

	following    []dbmodel.User
	followingErr error

	lastUserID      uuid.UUID
	lastPostID      uuid.UUID
	lastFollowerID  uuid.UUID
	lastFollowingID uuid.UUID
	lastLimit       int
}

func newFakeInteractionRepo() *fakeInteractionRepo {
	return &fakeInteractionRepo{
		userExistsResult:  true,
		existingUsers:     map[uuid.UUID]bool{},
		likeCreated:       true,
		unlikeDeleted:     true,
		bookmarkCreated:   true,
		unbookmarkDeleted: true,
		repostCreated:     true,
		undoRepostDeleted: true,
		followCreated:     true,
		unfollowDeleted:   true,
	}
}

func (f *fakeInteractionRepo) UserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	f.lastUserID = userID

	if f.userExistsErr != nil {
		return false, f.userExistsErr
	}

	if len(f.existingUsers) > 0 {
		return f.existingUsers[userID], nil
	}

	return f.userExistsResult, nil
}

func (f *fakeInteractionRepo) LikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error) {
	f.lastUserID = userID
	f.lastPostID = postID

	if f.likeErr != nil {
		return false, f.likeErr
	}

	return f.likeCreated, nil
}

func (f *fakeInteractionRepo) UnlikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error) {
	f.lastUserID = userID
	f.lastPostID = postID

	if f.unlikeErr != nil {
		return false, f.unlikeErr
	}

	return f.unlikeDeleted, nil
}

func (f *fakeInteractionRepo) BookmarkPost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error) {
	f.lastUserID = userID
	f.lastPostID = postID

	if f.bookmarkErr != nil {
		return false, f.bookmarkErr
	}

	return f.bookmarkCreated, nil
}

func (f *fakeInteractionRepo) UnbookmarkPost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error) {
	f.lastUserID = userID
	f.lastPostID = postID

	if f.unbookmarkErr != nil {
		return false, f.unbookmarkErr
	}

	return f.unbookmarkDeleted, nil
}

func (f *fakeInteractionRepo) RepostPost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error) {
	f.lastUserID = userID
	f.lastPostID = postID

	if f.repostErr != nil {
		return false, f.repostErr
	}

	return f.repostCreated, nil
}

func (f *fakeInteractionRepo) UndoRepost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error) {
	f.lastUserID = userID
	f.lastPostID = postID

	if f.undoRepostErr != nil {
		return false, f.undoRepostErr
	}

	return f.undoRepostDeleted, nil
}

func (f *fakeInteractionRepo) FollowUser(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) (bool, error) {
	f.lastFollowerID = followerID
	f.lastFollowingID = followingID

	if f.followErr != nil {
		return false, f.followErr
	}

	return f.followCreated, nil
}

func (f *fakeInteractionRepo) UnfollowUser(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) (bool, error) {
	f.lastFollowerID = followerID
	f.lastFollowingID = followingID

	if f.unfollowErr != nil {
		return false, f.unfollowErr
	}

	return f.unfollowDeleted, nil
}

func (f *fakeInteractionRepo) GetFollowers(ctx context.Context, userID uuid.UUID, limit int) ([]dbmodel.User, error) {
	f.lastUserID = userID
	f.lastLimit = limit

	if f.followersErr != nil {
		return nil, f.followersErr
	}

	return f.followers, nil
}

func (f *fakeInteractionRepo) GetFollowing(ctx context.Context, userID uuid.UUID, limit int) ([]dbmodel.User, error) {
	f.lastUserID = userID
	f.lastLimit = limit

	if f.followingErr != nil {
		return nil, f.followingErr
	}

	return f.following, nil
}

func newTestContentService(postRepo *fakePostRepo, interactionRepo *fakeInteractionRepo, publisher *fakeContentPublisher) ContentService {
	return NewContentService(postRepo, interactionRepo, publisher, zerolog.Nop())
}

func TestCreatePostSuccess(t *testing.T) {
	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	publisher := &fakeContentPublisher{}

	svc := newTestContentService(postRepo, interactionRepo, publisher)

	authorID := uuid.New()
	mediaID := uuid.NewString()

	res, err := svc.CreatePost(context.Background(), dto.CreatePostRequest{
		AuthorID: authorID.String(),
		Content:  "  hello threadly  ",
		MediaIDs: []string{mediaID},
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "hello threadly", res.Content)
	assert.Equal(t, "public", res.Visibility)
	assert.Equal(t, authorID.String(), res.AuthorID)

	require.NotNil(t, postRepo.createdPost)
	assert.Equal(t, "hello threadly", postRepo.createdPost.Content)
	assert.Equal(t, "public", postRepo.createdPost.Visibility)

	assert.Equal(t, postRepo.createdPost.ID, postRepo.attachedPostID)
	assert.Equal(t, authorID, postRepo.attachedUploader)
	assert.Equal(t, []string{mediaID}, postRepo.attachedMediaIDs)

	require.Len(t, publisher.published, 1)
	assert.Equal(t, messaging.EventPostCreated, publisher.published[0].routingKey)
	assert.Equal(t, messaging.EventPostCreated, publisher.published[0].event.Type)
	assert.Equal(t, authorID.String(), publisher.published[0].event.ActorID)
	assert.Equal(t, postRepo.createdPost.ID.String(), publisher.published[0].event.PostID)
}

func TestCreatePostInvalidContent(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.CreatePost(context.Background(), dto.CreatePostRequest{
		AuthorID: uuid.NewString(),
		Content:  "   ",
	})

	require.ErrorIs(t, err, ErrInvalidPostContent)
	assert.Nil(t, res)
}

func TestCreatePostInvalidAuthorID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.CreatePost(context.Background(), dto.CreatePostRequest{
		AuthorID: "bad-author-id",
		Content:  "hello",
	})

	require.ErrorIs(t, err, ErrInvalidAuthorID)
	assert.Nil(t, res)
}

func TestCreatePostRepositoryCreateError(t *testing.T) {
	expectedErr := errors.New("create post failed")

	postRepo := newFakePostRepo()
	postRepo.createErr = expectedErr

	svc := newTestContentService(postRepo, newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.CreatePost(context.Background(), dto.CreatePostRequest{
		AuthorID: uuid.NewString(),
		Content:  "hello",
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestCreatePostAttachMediaError(t *testing.T) {
	expectedErr := errors.New("attach media failed")

	postRepo := newFakePostRepo()
	postRepo.attachErr = expectedErr

	svc := newTestContentService(postRepo, newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.CreatePost(context.Background(), dto.CreatePostRequest{
		AuthorID: uuid.NewString(),
		Content:  "hello",
		MediaIDs: []string{uuid.NewString()},
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestCreatePostFindAfterCreateError(t *testing.T) {
	expectedErr := errors.New("find after create failed")

	postRepo := newFakePostRepo()
	postRepo.findErr = expectedErr

	svc := newTestContentService(postRepo, newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.CreatePost(context.Background(), dto.CreatePostRequest{
		AuthorID: uuid.NewString(),
		Content:  "hello",
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestGetPostSuccess(t *testing.T) {
	postRepo := newFakePostRepo()
	svc := newTestContentService(postRepo, newFakeInteractionRepo(), &fakeContentPublisher{})

	postID := uuid.New()
	authorID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "hello"))

	res, err := svc.GetPost(context.Background(), dto.GetPostRequest{
		PostID: postID.String(),
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, postID.String(), res.ID)
	assert.Equal(t, authorID.String(), res.AuthorID)
	assert.Equal(t, "hello", res.Content)
}

func TestGetPostInvalidPostID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.GetPost(context.Background(), dto.GetPostRequest{
		PostID: "bad-post-id",
	})

	require.ErrorIs(t, err, ErrInvalidPostID)
	assert.Nil(t, res)
}

func TestGetPostRepositoryError(t *testing.T) {
	postRepo := newFakePostRepo()
	svc := newTestContentService(postRepo, newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.GetPost(context.Background(), dto.GetPostRequest{
		PostID: uuid.NewString(),
	})

	require.ErrorIs(t, err, repository.ErrPostNotFound)
	assert.Nil(t, res)
}

func TestUpdatePostSuccess(t *testing.T) {
	postRepo := newFakePostRepo()
	publisher := &fakeContentPublisher{}

	svc := newTestContentService(postRepo, newFakeInteractionRepo(), publisher)

	postID := uuid.New()
	authorID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "old content"))

	mediaID := uuid.NewString()

	res, err := svc.UpdatePost(context.Background(), dto.UpdatePostRequest{
		PostID:      postID.String(),
		RequesterID: authorID.String(),
		Content:     "  updated content  ",
		MediaIDs:    []string{mediaID},
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "updated content", res.Content)
	assert.Equal(t, "public", res.Visibility)

	assert.Equal(t, postID.String(), postRepo.updatedPostID)
	assert.Equal(t, authorID.String(), postRepo.updatedRequester)
	assert.Equal(t, "updated content", postRepo.updatedContent)
	assert.Equal(t, "public", postRepo.updatedVisibility)

	assert.Equal(t, postID, postRepo.replacedPostID)
	assert.Equal(t, authorID, postRepo.replacedUploader)
	assert.Equal(t, []string{mediaID}, postRepo.replacedMediaIDs)

	require.Len(t, publisher.published, 1)
	assert.Equal(t, messaging.EventPostUpdated, publisher.published[0].routingKey)
	assert.Equal(t, messaging.EventPostUpdated, publisher.published[0].event.Type)
	assert.Equal(t, postID.String(), publisher.published[0].event.PostID)
}

func TestUpdatePostInvalidContent(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.UpdatePost(context.Background(), dto.UpdatePostRequest{
		PostID:      uuid.NewString(),
		RequesterID: uuid.NewString(),
		Content:     "   ",
	})

	require.ErrorIs(t, err, ErrInvalidPostContent)
	assert.Nil(t, res)
}

func TestUpdatePostInvalidPostID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.UpdatePost(context.Background(), dto.UpdatePostRequest{
		PostID:      "bad-post-id",
		RequesterID: uuid.NewString(),
		Content:     "updated",
	})

	require.ErrorIs(t, err, ErrInvalidPostID)
	assert.Nil(t, res)
}

func TestUpdatePostInvalidRequesterID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.UpdatePost(context.Background(), dto.UpdatePostRequest{
		PostID:      uuid.NewString(),
		RequesterID: "bad-requester-id",
		Content:     "updated",
	})

	require.ErrorIs(t, err, ErrInvalidAuthorID)
	assert.Nil(t, res)
}

func TestUpdatePostUpdateRepositoryError(t *testing.T) {
	expectedErr := errors.New("update failed")

	postRepo := newFakePostRepo()
	postRepo.updateErr = expectedErr

	svc := newTestContentService(postRepo, newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.UpdatePost(context.Background(), dto.UpdatePostRequest{
		PostID:      uuid.NewString(),
		RequesterID: uuid.NewString(),
		Content:     "updated",
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestUpdatePostReplaceMediaError(t *testing.T) {
	expectedErr := errors.New("replace media failed")

	postRepo := newFakePostRepo()

	postID := uuid.New()
	authorID := uuid.New()
	postRepo.seedPost(testPost(postID, authorID, "old"))

	postRepo.replaceErr = expectedErr

	svc := newTestContentService(postRepo, newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.UpdatePost(context.Background(), dto.UpdatePostRequest{
		PostID:      postID.String(),
		RequesterID: authorID.String(),
		Content:     "updated",
		MediaIDs:    []string{uuid.NewString()},
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestDeletePostSuccess(t *testing.T) {
	postRepo := newFakePostRepo()
	publisher := &fakeContentPublisher{}

	svc := newTestContentService(postRepo, newFakeInteractionRepo(), publisher)

	postID := uuid.New()
	authorID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "delete me"))

	err := svc.DeletePost(context.Background(), dto.DeletePostRequest{
		PostID:      postID.String(),
		RequesterID: authorID.String(),
	})

	require.NoError(t, err)

	assert.Equal(t, postID.String(), postRepo.deletedPostID)
	assert.Equal(t, authorID.String(), postRepo.deletedRequester)

	require.Len(t, publisher.published, 1)
	assert.Equal(t, messaging.EventPostDeleted, publisher.published[0].routingKey)
	assert.Equal(t, messaging.EventPostDeleted, publisher.published[0].event.Type)
	assert.Equal(t, postID.String(), publisher.published[0].event.PostID)
	assert.Equal(t, authorID.String(), publisher.published[0].event.AuthorID)
}

func TestDeletePostInvalidPostID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	err := svc.DeletePost(context.Background(), dto.DeletePostRequest{
		PostID:      "bad-post-id",
		RequesterID: uuid.NewString(),
	})

	require.ErrorIs(t, err, ErrInvalidPostID)
}

func TestDeletePostInvalidRequesterID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	err := svc.DeletePost(context.Background(), dto.DeletePostRequest{
		PostID:      uuid.NewString(),
		RequesterID: "bad-requester-id",
	})

	require.ErrorIs(t, err, ErrInvalidAuthorID)
}

func TestDeletePostFindError(t *testing.T) {
	postRepo := newFakePostRepo()
	svc := newTestContentService(postRepo, newFakeInteractionRepo(), &fakeContentPublisher{})

	err := svc.DeletePost(context.Background(), dto.DeletePostRequest{
		PostID:      uuid.NewString(),
		RequesterID: uuid.NewString(),
	})

	require.ErrorIs(t, err, repository.ErrPostNotFound)
}

func TestDeletePostDeleteError(t *testing.T) {
	expectedErr := errors.New("delete failed")

	postRepo := newFakePostRepo()

	postID := uuid.New()
	authorID := uuid.New()
	postRepo.seedPost(testPost(postID, authorID, "delete me"))
	postRepo.deleteErr = expectedErr

	svc := newTestContentService(postRepo, newFakeInteractionRepo(), &fakeContentPublisher{})

	err := svc.DeletePost(context.Background(), dto.DeletePostRequest{
		PostID:      postID.String(),
		RequesterID: authorID.String(),
	})

	require.ErrorIs(t, err, expectedErr)
}

func TestCreateReplySuccessPublishesEvent(t *testing.T) {
	postRepo := newFakePostRepo()
	publisher := &fakeContentPublisher{}

	svc := newTestContentService(postRepo, newFakeInteractionRepo(), publisher)

	parentPostID := uuid.New()
	parentAuthorID := uuid.New()
	replyAuthorID := uuid.New()

	postRepo.seedPost(testPost(parentPostID, parentAuthorID, "parent post"))

	res, err := svc.CreateReply(context.Background(), dto.CreateReplyRequest{
		AuthorID:      replyAuthorID.String(),
		ReplyToPostID: parentPostID.String(),
		Content:       "  reply content  ",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "reply content", res.Content)
	assert.Equal(t, parentPostID.String(), res.ReplyToPostID)

	require.NotNil(t, postRepo.createdPost)
	require.NotNil(t, postRepo.createdPost.ReplyToPostID)
	assert.Equal(t, parentPostID, *postRepo.createdPost.ReplyToPostID)
	assert.Equal(t, parentPostID.String(), postRepo.incrementPostID)

	require.Len(t, publisher.published, 1)
	assert.Equal(t, messaging.EventReplyCreated, publisher.published[0].routingKey)
	assert.Equal(t, messaging.EventReplyCreated, publisher.published[0].event.Type)
	assert.Equal(t, replyAuthorID.String(), publisher.published[0].event.ActorID)
	assert.Equal(t, parentAuthorID.String(), publisher.published[0].event.TargetUserID)
}

func TestCreateReplyToOwnPostDoesNotPublishNotificationEvent(t *testing.T) {
	postRepo := newFakePostRepo()
	publisher := &fakeContentPublisher{}

	svc := newTestContentService(postRepo, newFakeInteractionRepo(), publisher)

	parentPostID := uuid.New()
	authorID := uuid.New()

	postRepo.seedPost(testPost(parentPostID, authorID, "parent post"))

	res, err := svc.CreateReply(context.Background(), dto.CreateReplyRequest{
		AuthorID:      authorID.String(),
		ReplyToPostID: parentPostID.String(),
		Content:       "reply content",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Empty(t, publisher.published)
}

func TestCreateReplyInvalidContent(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.CreateReply(context.Background(), dto.CreateReplyRequest{
		AuthorID:      uuid.NewString(),
		ReplyToPostID: uuid.NewString(),
		Content:       "   ",
	})

	require.ErrorIs(t, err, ErrInvalidPostContent)
	assert.Nil(t, res)
}

func TestCreateReplyInvalidAuthorID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.CreateReply(context.Background(), dto.CreateReplyRequest{
		AuthorID:      "bad-author-id",
		ReplyToPostID: uuid.NewString(),
		Content:       "reply",
	})

	require.ErrorIs(t, err, ErrInvalidAuthorID)
	assert.Nil(t, res)
}

func TestCreateReplyInvalidReplyToPostID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.CreateReply(context.Background(), dto.CreateReplyRequest{
		AuthorID:      uuid.NewString(),
		ReplyToPostID: "bad-post-id",
		Content:       "reply",
	})

	require.ErrorIs(t, err, ErrInvalidPostID)
	assert.Nil(t, res)
}

func TestCreateReplyParentPostNotFound(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.CreateReply(context.Background(), dto.CreateReplyRequest{
		AuthorID:      uuid.NewString(),
		ReplyToPostID: uuid.NewString(),
		Content:       "reply",
	})

	require.ErrorIs(t, err, repository.ErrPostNotFound)
	assert.Nil(t, res)
}

func TestCreateReplyCreateError(t *testing.T) {
	expectedErr := errors.New("create reply failed")

	postRepo := newFakePostRepo()

	parentPostID := uuid.New()
	parentAuthorID := uuid.New()
	postRepo.seedPost(testPost(parentPostID, parentAuthorID, "parent"))
	postRepo.createErr = expectedErr

	svc := newTestContentService(postRepo, newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.CreateReply(context.Background(), dto.CreateReplyRequest{
		AuthorID:      uuid.NewString(),
		ReplyToPostID: parentPostID.String(),
		Content:       "reply",
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestCreateReplyIncrementReplyCountError(t *testing.T) {
	expectedErr := errors.New("increment reply count failed")

	postRepo := newFakePostRepo()

	parentPostID := uuid.New()
	parentAuthorID := uuid.New()
	postRepo.seedPost(testPost(parentPostID, parentAuthorID, "parent"))
	postRepo.incrementErr = expectedErr

	svc := newTestContentService(postRepo, newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.CreateReply(context.Background(), dto.CreateReplyRequest{
		AuthorID:      uuid.NewString(),
		ReplyToPostID: parentPostID.String(),
		Content:       "reply",
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestCreateReplyAttachMediaError(t *testing.T) {
	expectedErr := errors.New("attach reply media failed")

	postRepo := newFakePostRepo()

	parentPostID := uuid.New()
	parentAuthorID := uuid.New()
	postRepo.seedPost(testPost(parentPostID, parentAuthorID, "parent"))
	postRepo.attachErr = expectedErr

	svc := newTestContentService(postRepo, newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.CreateReply(context.Background(), dto.CreateReplyRequest{
		AuthorID:      uuid.NewString(),
		ReplyToPostID: parentPostID.String(),
		Content:       "reply",
		MediaIDs:      []string{uuid.NewString()},
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestGetRepliesSuccess(t *testing.T) {
	postRepo := newFakePostRepo()
	svc := newTestContentService(postRepo, newFakeInteractionRepo(), &fakeContentPublisher{})

	parentPostID := uuid.New()
	replyAuthorID := uuid.New()
	replyToPostID := parentPostID

	reply := testPost(uuid.New(), replyAuthorID, "reply one")
	reply.ReplyToPostID = &replyToPostID

	postRepo.replies[parentPostID.String()] = []dbmodel.Post{*reply}

	res, err := svc.GetReplies(context.Background(), dto.GetRepliesRequest{
		PostID: parentPostID.String(),
		Limit:  10,
	})

	require.NoError(t, err)
	require.Len(t, res, 1)

	assert.Equal(t, "reply one", res[0].Content)
	assert.Equal(t, parentPostID.String(), res[0].ReplyToPostID)
}

func TestGetRepliesInvalidPostID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.GetReplies(context.Background(), dto.GetRepliesRequest{
		PostID: "bad-post-id",
	})

	require.ErrorIs(t, err, ErrInvalidPostID)
	assert.Nil(t, res)
}

func TestGetRepliesRepositoryError(t *testing.T) {
	expectedErr := errors.New("find replies failed")

	postRepo := newFakePostRepo()
	postRepo.repliesErr = expectedErr

	svc := newTestContentService(postRepo, newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.GetReplies(context.Background(), dto.GetRepliesRequest{
		PostID: uuid.NewString(),
		Limit:  10,
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestLikePostSuccessPublishesEvent(t *testing.T) {
	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	publisher := &fakeContentPublisher{}

	svc := newTestContentService(postRepo, interactionRepo, publisher)

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.LikePost(context.Background(), dto.LikePostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, res.Success)
	assert.Equal(t, "post liked successfully", res.Message)
	assert.Equal(t, userID, interactionRepo.lastUserID)
	assert.Equal(t, postID, interactionRepo.lastPostID)

	require.Len(t, publisher.published, 1)
	assert.Equal(t, messaging.EventPostLiked, publisher.published[0].routingKey)
	assert.Equal(t, messaging.EventPostLiked, publisher.published[0].event.Type)
	assert.Equal(t, userID.String(), publisher.published[0].event.ActorID)
	assert.Equal(t, postID.String(), publisher.published[0].event.PostID)
	assert.Equal(t, authorID.String(), publisher.published[0].event.AuthorID)
}

func TestLikeOwnPostDoesNotPublishEvent(t *testing.T) {
	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	publisher := &fakeContentPublisher{}

	svc := newTestContentService(postRepo, interactionRepo, publisher)

	postID := uuid.New()
	authorID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.LikePost(context.Background(), dto.LikePostRequest{
		UserID: authorID.String(),
		PostID: postID.String(),
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "post liked successfully", res.Message)
	assert.Empty(t, publisher.published)
}

func TestLikePostAlreadyLiked(t *testing.T) {
	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.likeCreated = false

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.LikePost(context.Background(), dto.LikePostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "post already liked", res.Message)
}

func TestLikePostInvalidUserID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.LikePost(context.Background(), dto.LikePostRequest{
		UserID: "bad-user-id",
		PostID: uuid.NewString(),
	})

	require.ErrorIs(t, err, ErrInvalidUserID)
	assert.Nil(t, res)
}

func TestLikePostInvalidPostID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.LikePost(context.Background(), dto.LikePostRequest{
		UserID: uuid.NewString(),
		PostID: "bad-post-id",
	})

	require.ErrorIs(t, err, ErrInvalidPostID)
	assert.Nil(t, res)
}

func TestLikePostPostNotFound(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.LikePost(context.Background(), dto.LikePostRequest{
		UserID: uuid.NewString(),
		PostID: uuid.NewString(),
	})

	require.ErrorIs(t, err, repository.ErrPostNotFound)
	assert.Nil(t, res)
}

func TestLikePostRepositoryError(t *testing.T) {
	expectedErr := errors.New("like failed")

	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.likeErr = expectedErr

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.LikePost(context.Background(), dto.LikePostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestUnlikePostSuccess(t *testing.T) {
	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.UnlikePost(context.Background(), dto.UnlikePostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "post unliked successfully", res.Message)
	assert.Equal(t, userID, interactionRepo.lastUserID)
	assert.Equal(t, postID, interactionRepo.lastPostID)
}

func TestUnlikePostWasNotLiked(t *testing.T) {
	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.unlikeDeleted = false

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.UnlikePost(context.Background(), dto.UnlikePostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.NoError(t, err)
	assert.Equal(t, "post was not liked", res.Message)
}

func TestUnlikePostRepositoryError(t *testing.T) {
	expectedErr := errors.New("unlike failed")

	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.unlikeErr = expectedErr

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.UnlikePost(context.Background(), dto.UnlikePostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestBookmarkPostSuccess(t *testing.T) {
	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.BookmarkPost(context.Background(), dto.BookmarkPostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.NoError(t, err)
	assert.Equal(t, "post bookmarked successfully", res.Message)
}

func TestBookmarkPostAlreadyBookmarked(t *testing.T) {
	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.bookmarkCreated = false

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.BookmarkPost(context.Background(), dto.BookmarkPostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.NoError(t, err)
	assert.Equal(t, "post already bookmarked", res.Message)
}

func TestBookmarkPostRepositoryError(t *testing.T) {
	expectedErr := errors.New("bookmark failed")

	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.bookmarkErr = expectedErr

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.BookmarkPost(context.Background(), dto.BookmarkPostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestUnbookmarkPostSuccess(t *testing.T) {
	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.UnbookmarkPost(context.Background(), dto.UnbookmarkPostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.NoError(t, err)
	assert.Equal(t, "post unbookmarked successfully", res.Message)
}

func TestUnbookmarkPostWasNotBookmarked(t *testing.T) {
	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.unbookmarkDeleted = false

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.UnbookmarkPost(context.Background(), dto.UnbookmarkPostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.NoError(t, err)
	assert.Equal(t, "post was not bookmarked", res.Message)
}

func TestRepostPostSuccessPublishesEvent(t *testing.T) {
	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	publisher := &fakeContentPublisher{}

	svc := newTestContentService(postRepo, interactionRepo, publisher)

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.RepostPost(context.Background(), dto.RepostPostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.NoError(t, err)
	assert.Equal(t, "post reposted successfully", res.Message)

	require.Len(t, publisher.published, 1)
	assert.Equal(t, messaging.EventPostReposted, publisher.published[0].routingKey)
	assert.Equal(t, messaging.EventPostReposted, publisher.published[0].event.Type)
	assert.Equal(t, userID.String(), publisher.published[0].event.ActorID)
	assert.Equal(t, authorID.String(), publisher.published[0].event.AuthorID)
}

func TestRepostOwnPostDoesNotPublishEvent(t *testing.T) {
	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	publisher := &fakeContentPublisher{}

	svc := newTestContentService(postRepo, interactionRepo, publisher)

	postID := uuid.New()
	authorID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.RepostPost(context.Background(), dto.RepostPostRequest{
		UserID: authorID.String(),
		PostID: postID.String(),
	})

	require.NoError(t, err)
	assert.Equal(t, "post reposted successfully", res.Message)
	assert.Empty(t, publisher.published)
}

func TestRepostPostAlreadyReposted(t *testing.T) {
	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.repostCreated = false

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.RepostPost(context.Background(), dto.RepostPostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.NoError(t, err)
	assert.Equal(t, "post already reposted", res.Message)
}

func TestRepostPostRepositoryError(t *testing.T) {
	expectedErr := errors.New("repost failed")

	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.repostErr = expectedErr

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.RepostPost(context.Background(), dto.RepostPostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestUndoRepostSuccess(t *testing.T) {
	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.UndoRepost(context.Background(), dto.UndoRepostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.NoError(t, err)
	assert.Equal(t, "repost removed successfully", res.Message)
}

func TestUndoRepostWasNotReposted(t *testing.T) {
	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.undoRepostDeleted = false

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()

	postRepo.seedPost(testPost(postID, authorID, "post"))

	res, err := svc.UndoRepost(context.Background(), dto.UndoRepostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.NoError(t, err)
	assert.Equal(t, "post was not reposted", res.Message)
}

func TestFollowUserSuccessPublishesEvent(t *testing.T) {
	interactionRepo := newFakeInteractionRepo()
	publisher := &fakeContentPublisher{}

	svc := newTestContentService(newFakePostRepo(), interactionRepo, publisher)

	followerID := uuid.New()
	followingID := uuid.New()

	res, err := svc.FollowUser(context.Background(), dto.FollowUserRequest{
		FollowerID:  followerID.String(),
		FollowingID: followingID.String(),
	})

	require.NoError(t, err)
	assert.Equal(t, "user followed successfully", res.Message)

	assert.Equal(t, followerID, interactionRepo.lastFollowerID)
	assert.Equal(t, followingID, interactionRepo.lastFollowingID)

	require.Len(t, publisher.published, 1)
	assert.Equal(t, messaging.EventUserFollowed, publisher.published[0].routingKey)
	assert.Equal(t, messaging.EventUserFollowed, publisher.published[0].event.Type)
	assert.Equal(t, followerID.String(), publisher.published[0].event.ActorID)
	assert.Equal(t, followingID.String(), publisher.published[0].event.TargetUserID)
}

func TestFollowUserInvalidFollowerID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.FollowUser(context.Background(), dto.FollowUserRequest{
		FollowerID:  "bad-follower-id",
		FollowingID: uuid.NewString(),
	})

	require.ErrorIs(t, err, ErrInvalidUserID)
	assert.Nil(t, res)
}

func TestFollowUserInvalidFollowingID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.FollowUser(context.Background(), dto.FollowUserRequest{
		FollowerID:  uuid.NewString(),
		FollowingID: "bad-following-id",
	})

	require.ErrorIs(t, err, ErrInvalidUserID)
	assert.Nil(t, res)
}

func TestFollowUserCannotFollowSelf(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	userID := uuid.NewString()

	res, err := svc.FollowUser(context.Background(), dto.FollowUserRequest{
		FollowerID:  userID,
		FollowingID: userID,
	})

	require.ErrorIs(t, err, ErrCannotFollowSelf)
	assert.Nil(t, res)
}

func TestFollowUserFollowerNotFound(t *testing.T) {
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.userExistsResult = false

	svc := newTestContentService(newFakePostRepo(), interactionRepo, &fakeContentPublisher{})

	res, err := svc.FollowUser(context.Background(), dto.FollowUserRequest{
		FollowerID:  uuid.NewString(),
		FollowingID: uuid.NewString(),
	})

	require.ErrorIs(t, err, repository.ErrUserNotFound)
	assert.Nil(t, res)
}

func TestFollowUserAlreadyFollowed(t *testing.T) {
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.followCreated = false

	svc := newTestContentService(newFakePostRepo(), interactionRepo, &fakeContentPublisher{})

	res, err := svc.FollowUser(context.Background(), dto.FollowUserRequest{
		FollowerID:  uuid.NewString(),
		FollowingID: uuid.NewString(),
	})

	require.NoError(t, err)
	assert.Equal(t, "user already followed", res.Message)
}

func TestFollowUserRepositoryError(t *testing.T) {
	expectedErr := errors.New("follow failed")

	interactionRepo := newFakeInteractionRepo()
	interactionRepo.followErr = expectedErr

	svc := newTestContentService(newFakePostRepo(), interactionRepo, &fakeContentPublisher{})

	res, err := svc.FollowUser(context.Background(), dto.FollowUserRequest{
		FollowerID:  uuid.NewString(),
		FollowingID: uuid.NewString(),
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestUnfollowUserSuccessPublishesEvent(t *testing.T) {
	interactionRepo := newFakeInteractionRepo()
	publisher := &fakeContentPublisher{}

	svc := newTestContentService(newFakePostRepo(), interactionRepo, publisher)

	followerID := uuid.New()
	followingID := uuid.New()

	res, err := svc.UnfollowUser(context.Background(), dto.UnfollowUserRequest{
		FollowerID:  followerID.String(),
		FollowingID: followingID.String(),
	})

	require.NoError(t, err)
	assert.Equal(t, "user unfollowed successfully", res.Message)

	require.Len(t, publisher.published, 1)
	assert.Equal(t, messaging.EventUserUnfollowed, publisher.published[0].routingKey)
	assert.Equal(t, messaging.EventUserUnfollowed, publisher.published[0].event.Type)
}

func TestUnfollowUserWasNotFollowed(t *testing.T) {
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.unfollowDeleted = false

	svc := newTestContentService(newFakePostRepo(), interactionRepo, &fakeContentPublisher{})

	res, err := svc.UnfollowUser(context.Background(), dto.UnfollowUserRequest{
		FollowerID:  uuid.NewString(),
		FollowingID: uuid.NewString(),
	})

	require.NoError(t, err)
	assert.Equal(t, "user was not followed", res.Message)
}

func TestUnfollowUserCannotUnfollowSelf(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	userID := uuid.NewString()

	res, err := svc.UnfollowUser(context.Background(), dto.UnfollowUserRequest{
		FollowerID:  userID,
		FollowingID: userID,
	})

	require.ErrorIs(t, err, ErrCannotFollowSelf)
	assert.Nil(t, res)
}

func TestGetFollowersSuccess(t *testing.T) {
	interactionRepo := newFakeInteractionRepo()

	userID := uuid.New()
	followerID := uuid.New()

	avatarURL := "https://example.com/avatar.png"
	follower := testUser(followerID, "follower")
	follower.AvatarURL = &avatarURL
	follower.IsVerified = true

	interactionRepo.followers = []dbmodel.User{follower}

	svc := newTestContentService(newFakePostRepo(), interactionRepo, &fakeContentPublisher{})

	res, err := svc.GetFollowers(context.Background(), dto.GetFollowersRequest{
		UserID: userID.String(),
		Limit:  10,
	})

	require.NoError(t, err)
	require.Len(t, res, 1)

	assert.Equal(t, userID, interactionRepo.lastUserID)
	assert.Equal(t, 10, interactionRepo.lastLimit)
	assert.Equal(t, followerID.String(), res[0].ID)
	assert.Equal(t, "follower", res[0].Username)
	assert.Equal(t, avatarURL, res[0].AvatarURL)
	assert.True(t, res[0].IsVerified)
}

func TestGetFollowersInvalidUserID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.GetFollowers(context.Background(), dto.GetFollowersRequest{
		UserID: "bad-user-id",
	})

	require.ErrorIs(t, err, ErrInvalidUserID)
	assert.Nil(t, res)
}

func TestGetFollowersDefaultsInvalidLimit(t *testing.T) {
	interactionRepo := newFakeInteractionRepo()
	svc := newTestContentService(newFakePostRepo(), interactionRepo, &fakeContentPublisher{})

	_, err := svc.GetFollowers(context.Background(), dto.GetFollowersRequest{
		UserID: uuid.NewString(),
		Limit:  0,
	})

	require.NoError(t, err)
	assert.Equal(t, 20, interactionRepo.lastLimit)
}

func TestGetFollowersRepositoryError(t *testing.T) {
	expectedErr := errors.New("get followers failed")

	interactionRepo := newFakeInteractionRepo()
	interactionRepo.followersErr = expectedErr

	svc := newTestContentService(newFakePostRepo(), interactionRepo, &fakeContentPublisher{})

	res, err := svc.GetFollowers(context.Background(), dto.GetFollowersRequest{
		UserID: uuid.NewString(),
		Limit:  10,
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestGetFollowingSuccess(t *testing.T) {
	interactionRepo := newFakeInteractionRepo()

	userID := uuid.New()
	followingID := uuid.New()

	following := testUser(followingID, "following")
	interactionRepo.following = []dbmodel.User{following}

	svc := newTestContentService(newFakePostRepo(), interactionRepo, &fakeContentPublisher{})

	res, err := svc.GetFollowing(context.Background(), dto.GetFollowingRequest{
		UserID: userID.String(),
		Limit:  10,
	})

	require.NoError(t, err)
	require.Len(t, res, 1)

	assert.Equal(t, userID, interactionRepo.lastUserID)
	assert.Equal(t, 10, interactionRepo.lastLimit)
	assert.Equal(t, followingID.String(), res[0].ID)
	assert.Equal(t, "following", res[0].Username)
}

func TestGetFollowingInvalidUserID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.GetFollowing(context.Background(), dto.GetFollowingRequest{
		UserID: "bad-user-id",
	})

	require.ErrorIs(t, err, ErrInvalidUserID)
	assert.Nil(t, res)
}

func TestGetFollowingRepositoryError(t *testing.T) {
	expectedErr := errors.New("get following failed")

	interactionRepo := newFakeInteractionRepo()
	interactionRepo.followingErr = expectedErr

	svc := newTestContentService(newFakePostRepo(), interactionRepo, &fakeContentPublisher{})

	res, err := svc.GetFollowing(context.Background(), dto.GetFollowingRequest{
		UserID: uuid.NewString(),
		Limit:  10,
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestGetUserTimelineSuccess(t *testing.T) {
	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()

	userID := uuid.New()
	postID := uuid.New()

	post := testPost(postID, userID, "timeline post")
	repostedAt := time.Now().UTC()

	postRepo.findTimelineResult = []repository.UserTimelineItem{
		{
			Type:       "post",
			Post:       *post,
			RepostedAt: nil,
			SortAt:     post.CreatedAt,
		},
		{
			Type:       "repost",
			Post:       *post,
			RepostedAt: &repostedAt,
			SortAt:     repostedAt,
		},
	}

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	res, err := svc.GetUserTimeline(context.Background(), dto.GetUserTimelineRequest{
		UserID: userID.String(),
		Limit:  10,
	})

	require.NoError(t, err)
	require.Len(t, res, 2)

	assert.Equal(t, "post", res[0].Type)
	assert.Equal(t, "repost", res[1].Type)
	assert.Equal(t, repostedAt.Format(time.RFC3339), res[1].RepostedAt)
}

func TestGetUserTimelineInvalidUserID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.GetUserTimeline(context.Background(), dto.GetUserTimelineRequest{
		UserID: "bad-user-id",
	})

	require.ErrorIs(t, err, ErrInvalidUserID)
	assert.Nil(t, res)
}

func TestGetUserTimelineUserNotFound(t *testing.T) {
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.userExistsResult = false

	svc := newTestContentService(newFakePostRepo(), interactionRepo, &fakeContentPublisher{})

	res, err := svc.GetUserTimeline(context.Background(), dto.GetUserTimelineRequest{
		UserID: uuid.NewString(),
	})

	require.ErrorIs(t, err, repository.ErrUserNotFound)
	assert.Nil(t, res)
}

func TestGetUserTimelineRepositoryError(t *testing.T) {
	expectedErr := errors.New("timeline failed")

	postRepo := newFakePostRepo()
	postRepo.timelineErr = expectedErr

	svc := newTestContentService(postRepo, newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.GetUserTimeline(context.Background(), dto.GetUserTimelineRequest{
		UserID: uuid.NewString(),
		Limit:  10,
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestToMediaResponseWithDimensions(t *testing.T) {
	width := 640
	height := 480

	media := dbmodel.Media{
		ID:        uuid.New(),
		URL:       "https://example.com/image.png",
		MimeType:  "image/png",
		SizeBytes: 12345,
		Width:     &width,
		Height:    &height,
	}

	res := toMediaResponse(media)

	assert.Equal(t, media.ID.String(), res.ID)
	assert.Equal(t, "https://example.com/image.png", res.URL)
	assert.Equal(t, "image/png", res.MimeType)
	assert.Equal(t, int64(12345), res.SizeBytes)
	assert.Equal(t, 640, res.Width)
	assert.Equal(t, 480, res.Height)
}

func TestToMediaResponseWithoutDimensions(t *testing.T) {
	media := dbmodel.Media{
		ID:        uuid.New(),
		URL:       "https://example.com/image.png",
		MimeType:  "image/png",
		SizeBytes: 12345,
	}

	res := toMediaResponse(media)

	assert.Equal(t, 0, res.Width)
	assert.Equal(t, 0, res.Height)
}

func TestToPostResponseWithReplyMediaAndAvatar(t *testing.T) {
	authorID := uuid.New()
	postID := uuid.New()
	replyToPostID := uuid.New()
	mediaID := uuid.New()
	now := time.Now().UTC()

	avatarURL := "https://example.com/avatar.png"
	width := 300
	height := 200

	post := &dbmodel.Post{
		ID:            postID,
		AuthorID:      authorID,
		ReplyToPostID: &replyToPostID,
		Content:       "post with media",
		Visibility:    "public",
		LikeCount:     1,
		ReplyCount:    2,
		RepostCount:   3,
		BookmarkCount: 4,
		CreatedAt:     now,
		UpdatedAt:     now,
		Author: dbmodel.User{
			ID:          authorID,
			Username:    "author",
			DisplayName: "Author User",
			AvatarURL:   &avatarURL,
			IsVerified:  true,
		},
		Media: []dbmodel.Media{
			{
				ID:        mediaID,
				URL:       "https://example.com/media.png",
				MimeType:  "image/png",
				SizeBytes: 2048,
				Width:     &width,
				Height:    &height,
			},
		},
	}

	res := toPostResponse(post)

	require.NotNil(t, res)
	assert.Equal(t, postID.String(), res.ID)
	assert.Equal(t, replyToPostID.String(), res.ReplyToPostID)
	assert.Equal(t, avatarURL, res.Author.AvatarURL)
	require.Len(t, res.Media, 1)
	assert.Equal(t, mediaID.String(), res.Media[0].ID)
	assert.Equal(t, 300, res.Media[0].Width)
	assert.Equal(t, 200, res.Media[0].Height)
}

func TestPublishEventWithNilPublisherDoesNothing(t *testing.T) {
	svc := &contentService{
		eventPublisher: nil,
		log:            zerolog.Nop(),
	}

	svc.publishEvent(context.Background(), messaging.EventPostCreated, messaging.Event{
		EventID: uuid.NewString(),
		Type:    messaging.EventPostCreated,
	})
}

func TestPublishEventIgnoresPublisherError(t *testing.T) {
	publisher := &fakeContentPublisher{
		err: errors.New("rabbitmq down"),
	}

	svc := &contentService{
		eventPublisher: publisher,
		log:            zerolog.Nop(),
	}

	svc.publishEvent(context.Background(), messaging.EventPostCreated, messaging.Event{
		EventID: uuid.NewString(),
		Type:    messaging.EventPostCreated,
	})

	require.Len(t, publisher.published, 1)
	assert.Equal(t, messaging.EventPostCreated, publisher.published[0].routingKey)
}

func TestUnbookmarkPostInvalidUserID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.UnbookmarkPost(context.Background(), dto.UnbookmarkPostRequest{
		UserID: "bad-user-id",
		PostID: uuid.NewString(),
	})

	require.ErrorIs(t, err, ErrInvalidUserID)
	assert.Nil(t, res)
}

func TestUnbookmarkPostInvalidPostID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.UnbookmarkPost(context.Background(), dto.UnbookmarkPostRequest{
		UserID: uuid.NewString(),
		PostID: "bad-post-id",
	})

	require.ErrorIs(t, err, ErrInvalidPostID)
	assert.Nil(t, res)
}

func TestUnbookmarkPostPostNotFound(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.UnbookmarkPost(context.Background(), dto.UnbookmarkPostRequest{
		UserID: uuid.NewString(),
		PostID: uuid.NewString(),
	})

	require.ErrorIs(t, err, repository.ErrPostNotFound)
	assert.Nil(t, res)
}

func TestUnbookmarkPostRepositoryError(t *testing.T) {
	expectedErr := errors.New("unbookmark failed")

	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.unbookmarkErr = expectedErr

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()
	postRepo.seedPost(testPost(postID, authorID, "post"))

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	res, err := svc.UnbookmarkPost(context.Background(), dto.UnbookmarkPostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestUndoRepostInvalidUserID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.UndoRepost(context.Background(), dto.UndoRepostRequest{
		UserID: "bad-user-id",
		PostID: uuid.NewString(),
	})

	require.ErrorIs(t, err, ErrInvalidUserID)
	assert.Nil(t, res)
}

func TestUndoRepostInvalidPostID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.UndoRepost(context.Background(), dto.UndoRepostRequest{
		UserID: uuid.NewString(),
		PostID: "bad-post-id",
	})

	require.ErrorIs(t, err, ErrInvalidPostID)
	assert.Nil(t, res)
}

func TestUndoRepostPostNotFound(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.UndoRepost(context.Background(), dto.UndoRepostRequest{
		UserID: uuid.NewString(),
		PostID: uuid.NewString(),
	})

	require.ErrorIs(t, err, repository.ErrPostNotFound)
	assert.Nil(t, res)
}

func TestUndoRepostRepositoryError(t *testing.T) {
	expectedErr := errors.New("undo repost failed")

	postRepo := newFakePostRepo()
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.undoRepostErr = expectedErr

	postID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()
	postRepo.seedPost(testPost(postID, authorID, "post"))

	svc := newTestContentService(postRepo, interactionRepo, &fakeContentPublisher{})

	res, err := svc.UndoRepost(context.Background(), dto.UndoRepostRequest{
		UserID: userID.String(),
		PostID: postID.String(),
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestUnfollowUserInvalidFollowerID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.UnfollowUser(context.Background(), dto.UnfollowUserRequest{
		FollowerID:  "bad-follower-id",
		FollowingID: uuid.NewString(),
	})

	require.ErrorIs(t, err, ErrInvalidUserID)
	assert.Nil(t, res)
}

func TestUnfollowUserInvalidFollowingID(t *testing.T) {
	svc := newTestContentService(newFakePostRepo(), newFakeInteractionRepo(), &fakeContentPublisher{})

	res, err := svc.UnfollowUser(context.Background(), dto.UnfollowUserRequest{
		FollowerID:  uuid.NewString(),
		FollowingID: "bad-following-id",
	})

	require.ErrorIs(t, err, ErrInvalidUserID)
	assert.Nil(t, res)
}

func TestUnfollowUserFollowerNotFound(t *testing.T) {
	interactionRepo := newFakeInteractionRepo()
	interactionRepo.userExistsResult = false

	svc := newTestContentService(newFakePostRepo(), interactionRepo, &fakeContentPublisher{})

	res, err := svc.UnfollowUser(context.Background(), dto.UnfollowUserRequest{
		FollowerID:  uuid.NewString(),
		FollowingID: uuid.NewString(),
	})

	require.ErrorIs(t, err, repository.ErrUserNotFound)
	assert.Nil(t, res)
}

func TestUnfollowUserRepositoryError(t *testing.T) {
	expectedErr := errors.New("unfollow failed")

	interactionRepo := newFakeInteractionRepo()
	interactionRepo.unfollowErr = expectedErr

	svc := newTestContentService(newFakePostRepo(), interactionRepo, &fakeContentPublisher{})

	res, err := svc.UnfollowUser(context.Background(), dto.UnfollowUserRequest{
		FollowerID:  uuid.NewString(),
		FollowingID: uuid.NewString(),
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestGetFollowersUserExistsError(t *testing.T) {
	expectedErr := errors.New("user exists failed")

	interactionRepo := newFakeInteractionRepo()
	interactionRepo.userExistsErr = expectedErr

	svc := newTestContentService(newFakePostRepo(), interactionRepo, &fakeContentPublisher{})

	res, err := svc.GetFollowers(context.Background(), dto.GetFollowersRequest{
		UserID: uuid.NewString(),
		Limit:  10,
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestGetFollowingUserExistsError(t *testing.T) {
	expectedErr := errors.New("user exists failed")

	interactionRepo := newFakeInteractionRepo()
	interactionRepo.userExistsErr = expectedErr

	svc := newTestContentService(newFakePostRepo(), interactionRepo, &fakeContentPublisher{})

	res, err := svc.GetFollowing(context.Background(), dto.GetFollowingRequest{
		UserID: uuid.NewString(),
		Limit:  10,
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}
