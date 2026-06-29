package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Notification struct {
	ID      uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	EventID uuid.UUID `gorm:"column:event_id;type:uuid;not null;uniqueIndex"`

	RecipientID uuid.UUID  `gorm:"column:recipient_id;type:uuid;not null;index"`
	ActorID     *uuid.UUID `gorm:"column:actor_id;type:uuid;index"`

	Type       string     `gorm:"column:type;type:varchar(50);not null"`
	EntityType string     `gorm:"column:entity_type;type:varchar(50);not null"`
	EntityID   *uuid.UUID `gorm:"column:entity_id;type:uuid"`

	Payload datatypes.JSON `gorm:"column:payload;type:jsonb;not null"`

	ReadAt    *time.Time `gorm:"column:read_at"`
	CreatedAt time.Time  `gorm:"column:created_at;not null"`

	Recipient User  `gorm:"foreignKey:RecipientID"`
	Actor     *User `gorm:"foreignKey:ActorID"`
}

func (Notification) TableName() string {
	return "notifications"
}
