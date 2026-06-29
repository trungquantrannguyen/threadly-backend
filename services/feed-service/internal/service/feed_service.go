package service

import (
	"context"
	"errors"

	"github.com/rs/zerolog"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"github.com/trungquantrannguyen/threadly/services/feed-service/internal/repository"
)

var ErrUnauthorized = errors.New("Unauthorized")

type FeedService interface {
	GetHomeFeed(ctx context.Context, userID string, limit int, cursor string) ([]dbmodel.Post, string, error)
}

type feedService struct {
	feedRepo repository.FeedRepository
	log      zerolog.Logger
}

func NewFeedService(feedRepo repository.FeedRepository, log zerolog.Logger) FeedService {
	return &feedService{
		feedRepo: feedRepo,
		log:      log,
	}
}

func (s *feedService) GetHomeFeed(ctx context.Context, userID string, limit int, cursor string) ([]dbmodel.Post, string, error) {
	if userID == "" {
		return nil, "", ErrUnauthorized
	}

	return s.feedRepo.FindHomeFeed(ctx, userID, limit, cursor)
}
