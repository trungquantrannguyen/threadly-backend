package dto

type NotificationResponse struct {
	ID          string
	RecipientID string
	ActorID     string
	Type        string
	EntityType  string
	EntityID    string
	Payload     string
	ReadAt      string
	CreatedAt   string
	Actor       *NotificationActorResponse
}

type NotificationActorResponse struct {
	ID          string
	Username    string
	DisplayName string
	AvatarURL   string
	IsVerified  bool
}
