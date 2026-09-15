package model

// User 用户（根实体，跨家庭）。
type User struct {
	Base
	Email    string `gorm:"uniqueIndex;type:varchar(255);not null"`
	Name     string `gorm:"type:varchar(120);not null"`
	Password string `gorm:"type:varchar(255);not null"`
}

// TableName 指定表名。
func (User) TableName() string { return "users" }

// Family 家庭。
type Family struct {
	Base
	Name     string `gorm:"type:varchar(120);not null"`
	Timezone string `gorm:"type:varchar(64);default:Asia/Shanghai"`
	Currency string `gorm:"type:varchar(3);default:CNY"`
}

func (Family) TableName() string { return "families" }

// Member 家庭成员。
type Member struct {
	Base
	FamilyID string `gorm:"index;type:varchar(26);not null"`
	UserID   string `gorm:"index;type:varchar(26);not null"`
	Role     string `gorm:"type:varchar(20);not null"`
	Timezone string `gorm:"type:varchar(64);default:UTC"`
}

func (Member) TableName() string { return "family_members" }

// Cat 猫咪。
type Cat struct {
	Base
	FamilyID  string `gorm:"index;type:varchar(26);not null"`
	Name      string `gorm:"type:varchar(120);not null"`
	Gender    string `gorm:"type:varchar(10);not null"`
	Breed     string `gorm:"type:varchar(120)"`
	Birthday  string `gorm:"type:date"`
	Neutered  bool   `gorm:"default:false"`
	AvatarKey string `gorm:"type:varchar(255)"`
}

func (Cat) TableName() string { return "cats" }

// CatHealthProfile  健康档案（疾病、过敏、禁忌）。
type CatHealthProfile struct {
	Base
	CatID             string `gorm:"uniqueIndex;type:varchar(26);not null"`
	Diseases          string `gorm:"type:text"` // JSON 数组快照
	Allergies         string `gorm:"type:text"`
	Contraindications string `gorm:"type:text"`
}

func (CatHealthProfile) TableName() string { return "cat_health_profiles" }
