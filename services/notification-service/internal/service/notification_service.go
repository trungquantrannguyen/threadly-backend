package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"github.com/trungquantrannguyen/threadly/pkg/messaging"
	"github.com/trungquantrannguyen/threadly/services/notification-service/internal/dto"
	"github.com/trungquantrannguyen/threadly/services/notification-service/internal/repository"
	"gorm.io/datatypes"
)

var ErrInvalidNotificationEvent = errors.New("invalid notification event")

const (
	NotificationTypeUserFollowed = "user_followed"
	NotificationTypePostLiked    = "post_liked"
	NotificationTypePostReplied  = "post_replied"
	NotificationTypePostReposted = "post_reposted"

	EntityTypeUser = "user"
	EntityTypePost = "post"
)

type NotificationService interface {
	CreateFromEvent(ctx context.Context, event messaging.Event) error
	GetNotifications(ctx context.Context, userID string, limit int) ([]dto.NotificationResponse, error)
	GetUnreadNotificationCount(ctx context.Context, userID string) (int64, error)
	MarkNotificationRead(ctx context.Context, userID string, notificationID string) (bool, error)
	MarkAllNotificationsRead(ctx context.Context, userID string) (int64, error)
}

type notificationService struct {
	notificationRepo repository.NotificationRepository
}

func NewNotificationService(notificationRepo repository.NotificationRepository) NotificationService {
	return &notificationService{
		notificationRepo: notificationRepo,
	}
}

func (s *notificationService) CreateFromEvent(ctx context.Context, event messaging.Event) error {
	switch event.Type {
	case messaging.EventUserFollowed:
		return s.createFollowNotification(ctx, event)

	case messaging.EventPostLiked:
		return s.createPostNotification(ctx, event, NotificationTypePostLiked, "liked your post")

	case messaging.EventReplyCreated:
		return s.createReplyNotification(ctx, event)

	case messaging.EventPostReposted:
		return s.createPostNotification(ctx, event, NotificationTypePostReposted, "reposted your post")

	default:
		return nil
	}
}

func (s *notificationService) createFollowNotification(ctx context.Context, event messaging.Event) error {
	eventID, err := parseEventID(event)
	if err != nil {
		return err
	}

	if event.ActorID == "" || event.TargetUserID == "" {
		return ErrInvalidNotificationEvent
	}

	if event.ActorID == event.TargetUserID {
		return nil
	}

	actorID, err := uuid.Parse(event.ActorID)
	if err != nil {
		return ErrInvalidNotificationEvent
	}

	recipientID, err := uuid.Parse(event.TargetUserID)
	if err != nil {
		return ErrInvalidNotificationEvent
	}

	payload, err := json.Marshal(map[string]string{
		"message": "started following you",
	})
	if err != nil {
		return err
	}

	notification := &dbmodel.Notification{
		EventID:     eventID,
		RecipientID: recipientID,
		ActorID:     &actorID,
		Type:        NotificationTypeUserFollowed,
		EntityType:  EntityTypeUser,
		EntityID:    &actorID,
		Payload:     datatypes.JSON(payload),
		CreatedAt:   time.Now().UTC(),
	}

	return s.notificationRepo.Create(ctx, notification)
}

func (s *notificationService) createPostNotification(
	ctx context.Context,
	event messaging.Event,
	notificationType string,
	message string,
) error {
	eventID, err := parseEventID(event)
	if err != nil {
		return err
	}

	if event.ActorID == "" || event.AuthorID == "" || event.PostID == "" {
		return ErrInvalidNotificationEvent
	}

	if event.ActorID == event.AuthorID {
		return nil
	}

	actorID, err := uuid.Parse(event.ActorID)
	if err != nil {
		return ErrInvalidNotificationEvent
	}

	recipientID, err := uuid.Parse(event.AuthorID)
	if err != nil {
		return ErrInvalidNotificationEvent
	}

	postID, err := uuid.Parse(event.PostID)
	if err != nil {
		return ErrInvalidNotificationEvent
	}

	payload, err := json.Marshal(map[string]string{
		"message": message,
	})
	if err != nil {
		return err
	}

	notification := &dbmodel.Notification{
		EventID:     eventID,
		RecipientID: recipientID,
		ActorID:     &actorID,
		Type:        notificationType,
		EntityType:  EntityTypePost,
		EntityID:    &postID,
		Payload:     datatypes.JSON(payload),
		CreatedAt:   time.Now().UTC(),
	}

	return s.notificationRepo.Create(ctx, notification)
}

func (s *notificationService) createReplyNotification(ctx context.Context, event messaging.Event) error {
	eventID, err := parseEventID(event)
	if err != nil {
		return err
	}

	if event.ActorID == "" || event.TargetUserID == "" || event.PostID == "" {
		return ErrInvalidNotificationEvent
	}

	if event.ActorID == event.TargetUserID {
		return nil
	}

	actorID, err := uuid.Parse(event.ActorID)
	if err != nil {
		return ErrInvalidNotificationEvent
	}

	recipientID, err := uuid.Parse(event.TargetUserID)
	if err != nil {
		return ErrInvalidNotificationEvent
	}

	replyID, err := uuid.Parse(event.PostID)
	if err != nil {
		return ErrInvalidNotificationEvent
	}

	payload, err := json.Marshal(map[string]string{
		"message": "replied to your post",
	})
	if err != nil {
		return err
	}

	notification := &dbmodel.Notification{
		EventID:     eventID,
		RecipientID: recipientID,
		ActorID:     &actorID,
		Type:        NotificationTypePostReplied,
		EntityType:  EntityTypePost,
		EntityID:    &replyID,
		Payload:     datatypes.JSON(payload),
		CreatedAt:   time.Now().UTC(),
	}

	return s.notificationRepo.Create(ctx, notification)
}

func (s *notificationService) GetUnreadNotificationCount(ctx context.Context, userID string) (int64, error) {
	recipientID, err := uuid.Parse(userID)
	if err != nil {
		return 0, ErrInvalidNotificationEvent
	}

	return s.notificationRepo.CountUnreadByRecipientID(ctx, recipientID)
}

func (s *notificationService) GetNotifications(ctx context.Context, userID string, limit int) ([]dto.NotificationResponse, error) {
	recipientID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidNotificationEvent
	}

	if limit <= 0 || limit > 50 {
		limit = 20
	}

	notifications, err := s.notificationRepo.FindByRecipientID(ctx, recipientID, limit)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.NotificationResponse, 0, len(notifications))
	for _, notification := range notifications {
		responses = append(responses, toNotificationResponse(notification))
	}

	return responses, nil
}

func (s *notificationService) MarkNotificationRead(ctx context.Context, userID string, notificationID string) (bool, error) {
	recipientID, err := uuid.Parse(userID)
	if err != nil {
		return false, ErrInvalidNotificationEvent
	}

	id, err := uuid.Parse(notificationID)
	if err != nil {
		return false, ErrInvalidNotificationEvent
	}

	return s.notificationRepo.MarkAsRead(ctx, id, recipientID)
}

func (s *notificationService) MarkAllNotificationsRead(ctx context.Context, userID string) (int64, error) {
	recipientID, err := uuid.Parse(userID)
	if err != nil {
		return 0, ErrInvalidNotificationEvent
	}

	return s.notificationRepo.MarkAllAsRead(ctx, recipientID)
}

func toNotificationResponse(notification dbmodel.Notification) dto.NotificationResponse {
	actorID := ""
	if notification.ActorID != nil {
		actorID = notification.ActorID.String()
	}

	entityID := ""
	if notification.EntityID != nil {
		entityID = notification.EntityID.String()
	}

	readAt := ""
	if notification.ReadAt != nil {
		readAt = notification.ReadAt.Format(time.RFC3339)
	}

	var actor *dto.NotificationActorResponse
	if notification.Actor != nil {
		avatarURL := ""
		if notification.Actor.AvatarURL != nil {
			avatarURL = *notification.Actor.AvatarURL
		}

		actor = &dto.NotificationActorResponse{
			ID:          notification.Actor.ID.String(),
			Username:    notification.Actor.Username,
			DisplayName: notification.Actor.DisplayName,
			AvatarURL:   avatarURL,
			IsVerified:  notification.Actor.IsVerified,
		}
	}

	return dto.NotificationResponse{
		ID:          notification.ID.String(),
		RecipientID: notification.RecipientID.String(),
		ActorID:     actorID,
		Type:        notification.Type,
		EntityType:  notification.EntityType,
		EntityID:    entityID,
		Payload:     string(notification.Payload),
		ReadAt:      readAt,
		CreatedAt:   notification.CreatedAt.Format(time.RFC3339),
		Actor:       actor,
	}
}

func parseEventID(event messaging.Event) (uuid.UUID, error) {
	if event.EventID == "" {
		return uuid.Nil, ErrInvalidNotificationEvent
	}

	eventID, err := uuid.Parse(event.EventID)
	if err != nil {
		return uuid.Nil, ErrInvalidNotificationEvent
	}

	return eventID, nil
}
