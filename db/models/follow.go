package model

import (
	"time"

	"github.com/google/uuid"
)

type Follow struct {
	FollowerID  uuid.UUID `gorm:"column:follower_id;type:uuid;primaryKey"`
	FollowingID uuid.UUID `gorm:"column:following_id;type:uuid;primaryKey"`

	CreatedAt time.Time `gorm:"column:created_at;not null"`

	Follower  User `gorm:"foreignKey:FollowerID"`
	Following User `gorm:"foreignKey:FollowingID"`
}

func (Follow) TableName() string {
	return "follows"
}
