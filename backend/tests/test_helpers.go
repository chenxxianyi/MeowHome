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
// 复用 cmd/migrate 的 loadMigration 逻辑，自动剔除 -- +migrate Down 段，
// 避免早期文件（001/002）中嵌入的回滚语句被执行。
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
		// 仅执行 *.up.sql 文件，并剔除其中的 Down 段
		if filepath.Ext(f.Name()) == ".sql" && strings.HasSuffix(f.Name(), ".up.sql") {
			body, err := loadMigration(filepath.Join(dir, f.Name()), false)
			if err != nil {
				return fmt.Errorf("read migration %s: %w", f.Name(), err)
			}
			if strings.TrimSpace(body) == "" {
				t.Logf("skip    %s（无有效语句）", f.Name())
				continue
			}
			if err := db.Exec(body).Error; err != nil {
				return fmt.Errorf("apply migration %s: %w", f.Name(), err)
			}
			t.Logf("applied %s", f.Name())
		}
	}
	_ = context.Background() // 保留 context 供后续迁移使用
	return nil
}

// loadMigration 读取迁移文件并剔除 sql-migrate 标记。
// up 方向遇到 `-- +migrate Down` 即截断，与 cmd/migrate/main.go 的
// 行为保持一致，避免测试误执行早期文件（001/002）中嵌入的回滚语句。
func loadMigration(path string, down bool) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var out []string
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "-- +migrate") {
			if !down && strings.Contains(strings.ToLower(trimmed), "down") {
				break
			}
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n"), nil
}

// RollbackTestDB 清理测试数据库并关闭连接。
func RollbackTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_ = db.Exec("DROP DATABASE IF EXISTS meowhome_test")
	require.NoError(t, sqlDB.Close())
}
