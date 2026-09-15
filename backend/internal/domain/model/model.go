// Package model 定义领域模型。
package model

import (
	"time"

	"github.com/meowhome/backend/internal/platform/id"
)

// Base 通用基础字段。
// DeletedAt 用指针：MySQL 8 严格模式（NO_ZERO_DATE）会拒绝零值时间，
// 非指针 time.Time 在插入时会被写成 '0000-00-00' 而报错。
type Base struct {
	ID        string     `gorm:"primaryKey;type:varchar(26)"`
	CreatedBy string     `gorm:"type:varchar(26);not null"`
	CreatedAt time.Time  `gorm:"autoCreateTime;not null"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime;not null"`
	DeletedAt *time.Time `gorm:"index"`
}

// FamilyScope 家庭授权上下文。
type FamilyScope struct {
	UserID      string
	FamilyID    string
	MemberID    string
	Role        string
	Permissions []string
	Timezone    string
}

// NewBase 生成带家庭归属的基础记录。
func NewBase(familyID, userID string) Base {
	return Base{
		ID:        id.ULIDGenerator{}.New(),
		CreatedBy: userID,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}
