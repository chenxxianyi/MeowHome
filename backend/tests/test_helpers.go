// Package tests 提供测试基础设施。
package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// SetupTestDB 在测试中连接 MySQL 测试库并返回可连接的 *gorm.DB。
// 需要在 MYSQL_TEST_DSN 环境变量中配置连接串；未设置时跳过整个集成测试套件。
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN not set, skipping integration tests")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "failed to connect test DB")

	require.NoError(t, db.Exec("CREATE DATABASE IF NOT EXISTS meowhome_test").Error)
	require.NoError(t, db.Exec("USE meowhome_test").Error)

	require.NoError(t, applyMigrations(db, t))
	return db
}

// applyMigrations 按文件名顺序执行 migrations/ 下的 .up.sql 迁移。
func applyMigrations(db *gorm.DB, t *testing.T) error {
	dir := "../migrations"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		dir = filepath.Join("..", "migrations")
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() || len(f.Name()) < 3 {
			continue
		}
		// 仅执行 *.up.sql 文件
		if filepath.Ext(f.Name()) == ".sql" && strings.HasSuffix(f.Name(), ".up.sql") {
			data, err := os.ReadFile(filepath.Join(dir, f.Name()))
			if err != nil {
				return fmt.Errorf("read migration %s: %w", f.Name(), err)
			}
			if err := db.Exec(string(data)).Error; err != nil {
				return fmt.Errorf("apply migration %s: %w", f.Name(), err)
			}
		}
	}
	_ = context.Background() // 保留 context 供后续迁移使用
	return nil
}

// RollbackTestDB 清理测试数据库并关闭连接。
func RollbackTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_ = db.Exec("DROP DATABASE IF EXISTS meowhome_test")
	require.NoError(t, sqlDB.Close())
}
