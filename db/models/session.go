package model

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`

	UserID uuid.UUID `gorm:"column:user_id;type:uuid;not null;index"`

	RefreshTokenHash string  `gorm:"column:refresh_token_hash;type:text;uniqueIndex;not null"`
	UserAgent        *string `gorm:"column:user_agent;type:text"`
	IPAddress        *string `gorm:"column:ip_address;type:inet"`

	ExpiresAt time.Time  `gorm:"column:expires_at;not null;index"`
	RevokedAt *time.Time `gorm:"column:revoked_at;index"`
	CreatedAt time.Time  `gorm:"column:created_at;not null"`

	User User `gorm:"foreignKey:UserID"`
}

func (Session) TableName() string {
	return "sessions"
}
