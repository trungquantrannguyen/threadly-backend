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
)

type fakeFeedRepository struct {
	posts      []dbmodel.Post
	nextCursor string
	err        error

	calledUserID string
	calledLimit  int
	calledCursor string
}

func (f *fakeFeedRepository) FindHomeFeed(ctx context.Context, userID string, limit int, cursor string) ([]dbmodel.Post, string, error) {
	f.calledUserID = userID
	f.calledLimit = limit
	f.calledCursor = cursor

	if f.err != nil {
		return nil, "", f.err
	}

	return f.posts, f.nextCursor, nil
}

func TestGetHomeFeedSuccess(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()

	repo := &fakeFeedRepository{
		posts: []dbmodel.Post{
			{
				ID:         postID,
				AuthorID:   userID,
				Content:    "hello threadly",
				Visibility: "public",
				CreatedAt:  time.Now().UTC(),
				UpdatedAt:  time.Now().UTC(),
			},
		},
		nextCursor: "next-cursor",
	}

	svc := NewFeedService(repo, zerolog.Nop())

	posts, nextCursor, err := svc.GetHomeFeed(context.Background(), userID.String(), 20, "cursor-1")

	require.NoError(t, err)
	require.Len(t, posts, 1)

	assert.Equal(t, postID, posts[0].ID)
	assert.Equal(t, "next-cursor", nextCursor)
	assert.Equal(t, userID.String(), repo.calledUserID)
	assert.Equal(t, 20, repo.calledLimit)
	assert.Equal(t, "cursor-1", repo.calledCursor)
}

func TestGetHomeFeedUnauthorizedWhenUserIDEmpty(t *testing.T) {
	repo := &fakeFeedRepository{}
	svc := NewFeedService(repo, zerolog.Nop())

	posts, nextCursor, err := svc.GetHomeFeed(context.Background(), "", 20, "")

	require.ErrorIs(t, err, ErrUnauthorized)
	assert.Nil(t, posts)
	assert.Empty(t, nextCursor)
	assert.Empty(t, repo.calledUserID)
}

func TestGetHomeFeedRepositoryError(t *testing.T) {
	expectedErr := errors.New("database failed")

	repo := &fakeFeedRepository{
		err: expectedErr,
	}

	svc := NewFeedService(repo, zerolog.Nop())

	posts, nextCursor, err := svc.GetHomeFeed(context.Background(), uuid.NewString(), 20, "")

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, posts)
	assert.Empty(t, nextCursor)
}
