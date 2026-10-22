// Package mysql 实现 GORM 连接与 Repository 基类。
package mysql

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/meowhome/backend/internal/platform/config"
)

const RequiredSchemaVersion = 15

// New 创建 GORM 连接。
func New(cfg config.MySQL) (*gorm.DB, error) {
	dsn := buildDSN(cfg)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("mysql: connect: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("mysql: pool: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	return db, nil
}

func buildDSN(cfg config.MySQL) string {
	tls := "false"
	if cfg.TLS {
		tls = "true"
	}
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, tls,
	)
}

// Healthy 数据库可用性检查。
func Healthy(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	return sqlDB.PingContext(ctx)
}

// CheckSchemaVersion reports a clear readiness error when migrations have not reached this build's schema.
func CheckSchemaVersion(db *gorm.DB) error {
	var current int
	err := db.Raw("SELECT COALESCE(MAX(CAST(SUBSTRING_INDEX(version, '_', 1) AS UNSIGNED)), 0) FROM schema_migrations").Scan(&current).Error
	if err != nil {
		return fmt.Errorf("migration history unavailable; run `go run ./cmd/migrate`: %w", err)
	}
	if current < RequiredSchemaVersion {
		return fmt.Errorf("database schema version %03d is behind required version %03d; run `go run ./cmd/migrate`", current, RequiredSchemaVersion)
	}
	return nil
}
