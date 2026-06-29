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
}
