package model

import "time"

// DailyRecord 日常记录事件头。
type DailyRecord struct {
	Base
	FamilyID       string    `gorm:"index;type:varchar(26);not null"`
	CatIDs         []string  `gorm:"serializer:json;type:text"`
	RecordType     string    `gorm:"type:varchar(32);not null"`
	OccurredAt     time.Time `gorm:"index;not null"`
	Source         string    `gorm:"type:varchar(20);default:manual"`
	Severity       string    `gorm:"type:varchar(20);default:normal"`
	Title          string    `gorm:"type:varchar(255)"`
	Note           string    `gorm:"type:text"`
	Payload        string    `gorm:"type:text"` // 类型专属字段的 JSON（食量/体重/性状等）
	MedicalVisitID string    `gorm:"type:varchar(26)"`
}

func (DailyRecord) TableName() string { return "daily_records" }
