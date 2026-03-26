package migration

import (
	"fmt"

	"goflow/internal/model"
	"goflow/internal/pkg/logger"

	"gorm.io/gorm"
)

// AutoMigrate 自动迁移数据库表结构，返回 error 让调用方统一处理
func AutoMigrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		&model.User{},
		&model.Admin{},
		&model.Product{},
	)
	if err != nil {
		return fmt.Errorf("auto migrate failed: %w", err)
	}
	logger.Log.Info("database migration completed")
	return nil
}
