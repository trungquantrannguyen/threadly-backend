package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`

	Email        string `gorm:"type:citext;uniqueIndex;not null"`
	Username     string `gorm:"type:citext;uniqueIndex;not null"`
	PasswordHash string `gorm:"column:password_hash;type:text;not null"`

	DisplayName string  `gorm:"column:display_name;type:varchar(80);not null"`
	Bio         *string `gorm:"type:varchar(280)"`
	AvatarURL   *string `gorm:"column:avatar_url;type:text"`
	BannerURL   *string `gorm:"column:banner_url;type:text"`
	Location    *string `gorm:"type:varchar(100)"`
	WebsiteURL  *string `gorm:"column:website_url;type:text"`

	IsVerified bool `gorm:"column:is_verified;not null;default:false"`

	FollowerCount  int `gorm:"column:follower_count;not null;default:0"`
	FollowingCount int `gorm:"column:following_count;not null;default:0"`
	PostCount      int `gorm:"column:post_count;not null;default:0"`

	CreatedAt time.Time      `gorm:"column:created_at;not null"`
	UpdatedAt time.Time      `gorm:"column:updated_at;not null"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (User) TableName() string {
	return "users"
}
