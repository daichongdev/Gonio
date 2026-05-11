package model

// User 用户模型
type User struct {
	BaseModel
	Username string `json:"username" gorm:"type:varchar(64);uniqueIndex:idx_username;not null"`
	Password string `json:"-" gorm:"type:varchar(255);not null"`
	Nickname string `json:"nickname" gorm:"type:varchar(64)"`
	Avatar   string `json:"avatar" gorm:"type:varchar(255)"`
	Status   int    `json:"status" gorm:"default:1;index:idx_status;comment:1-正常 0-禁用"`
}
