// Package tests 提供测试基础设施。
package tests

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB 在测试中连接 MySQL 测试库并返回可连接的 *gorm.DB。
// 显式设置 MYSQL_TEST_DSN 或 MYSQL_TEST_ENV_FILE；均未设置时跳过。
// 只复用服务器凭据，每个测试创建自己的临时库，不使用配置中的业务库名。
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	cfg := testMySQLConfig(t)
	cfg.DBName = ""
	cfg.MultiStatements = true
	cfg.ParseTime = true
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 30 * time.Second
	cfg.WriteTimeout = 30 * time.Second
	admin, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("test database server unavailable")
	}
	adminSQL, err := admin.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = adminSQL.Close() })
	var server struct {
		Version string
		SQLMode string
	}
	require.NoError(t, admin.Raw("SELECT VERSION() AS version, @@SESSION.sql_mode AS sql_mode").Scan(&server).Error)
	t.Logf("MySQL version=%s sql_mode=%s", server.Version, server.SQLMode)
	databaseName := "meowhome_agent_test_" + strings.ToLower(ulid.Make().String())
	require.NoError(t, admin.Exec("CREATE DATABASE `"+databaseName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error)
	t.Logf("created isolated database %s", databaseName)
	t.Cleanup(func() {
		if !strings.HasPrefix(databaseName, "meowhome_agent_test_") || len(databaseName) != len("meowhome_agent_test_")+26 {
			t.Error("invalid owned test database name")
			return
		}
		if err := admin.Exec("DROP DATABASE `" + databaseName + "`").Error; err != nil {
			t.Error("failed to clean owned test database")
			return
		}
		t.Logf("cleaned isolated database %s", databaseName)
	})
	cfg.DBName = databaseName
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("failed to open isolated test database")
	}
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, applyMigrations(db, t))
	return db
}

func testMySQLConfig(t *testing.T) *driver.Config {
	t.Helper()
	if dsn := os.Getenv("MYSQL_TEST_DSN"); dsn != "" {
		cfg, err := driver.ParseDSN(dsn)
		if err != nil {
			t.Fatal("invalid MYSQL_TEST_DSN")
		}
		return cfg
	}
	path := os.Getenv("MYSQL_TEST_ENV_FILE")
	if path == "" {
		t.Skip("MYSQL_TEST_DSN / MYSQL_TEST_ENV_FILE not set, skipping integration tests")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal("cannot read MYSQL_TEST_ENV_FILE")
	}
	defer f.Close()
	// Match the application's dotenv quoting/comments without loading AI or auth secrets.
	values := map[string]string{"MYSQL_HOST": "127.0.0.1", "MYSQL_PORT": "3306", "MYSQL_TLS": "false"}
	allowed := map[string]bool{"MYSQL_HOST": true, "MYSQL_PORT": true, "MYSQL_USER": true, "MYSQL_PASSWORD": true, "MYSQL_TLS": true}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !found || !allowed[key] {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1]
		} else if i := strings.Index(value, " #"); i >= 0 {
			value = strings.TrimSpace(value[:i])
		}
		values[key] = value
	}
	if scanner.Err() != nil {
		t.Fatal("cannot parse MYSQL_TEST_ENV_FILE")
	}
	for key := range allowed {
		if value, exists := os.LookupEnv(key); exists {
			values[key] = value
		}
	}
	if values["MYSQL_USER"] == "" || values["MYSQL_HOST"] == "" {
		t.Fatal("test database user and host must be configured")
	}
	tls, err := strconv.ParseBool(values["MYSQL_TLS"])
	if err != nil {
		t.Fatal("invalid test database TLS setting")
	}
	cfg := driver.NewConfig()
	cfg.User, cfg.Passwd = values["MYSQL_USER"], values["MYSQL_PASSWORD"]
	cfg.Net, cfg.Addr = "tcp", net.JoinHostPort(values["MYSQL_HOST"], values["MYSQL_PORT"])
	cfg.Params = map[string]string{"charset": "utf8mb4"}
	cfg.Loc = time.Local
	if tls {
		cfg.TLSConfig = "true"
	}
	return cfg
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
	require.NoError(t, sqlDB.Close())
}
