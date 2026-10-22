// Package main 提供数据库迁移执行器。
//
// 用法（在 backend/ 目录下执行，配置取自 .env）：
//
//	go run ./cmd/migrate                  # 执行未应用的 up 迁移
//	go run ./cmd/migrate -down            # 只回滚最近一次已应用迁移
//	go run ./cmd/migrate -baseline 015    # 为已核验的旧数据库登记版本，不执行 SQL
//	go run ./cmd/migrate -dir migrations -dry # 预览迁移文件，不连接数据库
//
// 注意：本项目 001/002 的 .up.sql 文件里同时含有 Up 与 Down 两段
// （以 `-- +migrate Down` 分隔）。直接把整个文件喂给 mysql 客户端会
// 先建表再删表，因此这里显式在 Down 标记处截断。
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"github.com/meowhome/backend/internal/platform/config"
)

func main() {
	down := flag.Bool("down", false, "执行 down 迁移（逆序）")
	dir := flag.String("dir", "migrations", "迁移文件目录")
	dry := flag.Bool("dry", false, "只打印将执行的语句，不落库")
	baseline := flag.String("baseline", "", "将已有数据库标记为指定版本（如 015），不执行迁移")
	flag.Parse()
	if *down && *baseline != "" {
		fatal("-down 与 -baseline 不能同时使用")
	}
	if *dry && *baseline != "" {
		fatal("-dry 与 -baseline 不能同时使用")
	}

	cfg, err := config.Load()
	if err != nil {
		fatal("加载配置失败: %v", err)
	}

	files, err := listMigrations(*dir)
	if err != nil {
		fatal("扫描迁移目录失败: %v", err)
	}
	if len(files) == 0 {
		fatal("在 %s 下未找到 *.up.sql", *dir)
	}
	if *dry {
		if *down {
			fatal("-down -dry 无法在不读取迁移账本的情况下确定最近版本")
		}
		for _, migration := range files {
			path := migration.UpPath
			body, loadErr := loadMigration(path, false)
			if loadErr != nil {
				fatal("%s: 读取失败: %v", filepath.Base(path), loadErr)
			}
			fmt.Printf("dry-run %s\n%s\n", filepath.Base(path), body)
		}
		return
	}

	var db *sql.DB
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&multiStatements=true&tls=%t",
		cfg.MySQL.User, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database, cfg.MySQL.TLS,
	)
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		fatal("连接数据库失败: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		fatal("数据库不可达 (%s:%s/%s): %v",
			cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database, err)
	}
	if err := ensureMigrationTable(db); err != nil {
		fatal("初始化迁移账本失败: %v", err)
	}
	if *baseline != "" {
		if err := baselineMigrations(db, files, *baseline); err != nil {
			fatal("设置 baseline 失败: %v", err)
		}
		return
	}
	if *down {
		if err := rollbackLatest(db, files); err != nil {
			fatal("回滚迁移失败: %v", err)
		}
		return
	}
	if err := applyPending(db, files); err != nil {
		fatal("应用迁移失败: %v", err)
	}
	fmt.Println("迁移完成")
}

type migrationFile struct {
	Version  int
	Name     string
	UpPath   string
	DownPath string
}

func listMigrations(dir string) ([]migrationFile, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	files := make([]migrationFile, 0, len(paths))
	for _, upPath := range paths {
		name := filepath.Base(upPath)
		version, err := migrationNumber(name)
		if err != nil {
			return nil, err
		}
		files = append(files, migrationFile{
			Version:  version,
			Name:     name,
			UpPath:   upPath,
			DownPath: filepath.Join(dir, strings.TrimSuffix(name, ".up.sql")+".down.sql"),
		})
	}
	return files, nil
}

func migrationNumber(name string) (int, error) {
	separator := strings.IndexByte(name, '_')
	if separator <= 0 {
		return 0, fmt.Errorf("迁移文件名缺少版本前缀: %s", name)
	}
	version, err := strconv.Atoi(name[:separator])
	if err != nil || version < 1 {
		return 0, fmt.Errorf("迁移文件版本无效: %s", name)
	}
	return version, nil
}

func ensureMigrationTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) NOT NULL PRIMARY KEY,
		applied_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	return err
}

func applyPending(db *sql.DB, files []migrationFile) error {
	for _, migration := range files {
		var exists int
		if err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)", migration.Name).Scan(&exists); err != nil {
			return fmt.Errorf("检查 %s 状态失败: %w", migration.Name, err)
		}
		if exists != 0 {
			fmt.Printf("skip    %s（已应用）\n", migration.Name)
			continue
		}
		body, err := loadMigration(migration.UpPath, false)
		if err != nil {
			return fmt.Errorf("%s 读取失败: %w", migration.Name, err)
		}
		if strings.TrimSpace(body) != "" {
			if _, err := db.Exec(body); err != nil {
				return fmt.Errorf("%s 执行失败: %w", migration.Name, err)
			}
		}
		if _, err := db.Exec("INSERT INTO schema_migrations(version) VALUES (?)", migration.Name); err != nil {
			return fmt.Errorf("%s 记录版本失败: %w", migration.Name, err)
		}
		fmt.Printf("applied %s\n", migration.Name)
	}
	return nil
}

func rollbackLatest(db *sql.DB, files []migrationFile) error {
	var version string
	if err := db.QueryRow("SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("没有已应用的迁移可回滚")
		}
		return err
	}
	for _, migration := range files {
		if migration.Name != version {
			continue
		}
		body, err := loadMigration(migration.DownPath, true)
		if err != nil {
			return fmt.Errorf("%s 读取失败: %w", filepath.Base(migration.DownPath), err)
		}
		if strings.TrimSpace(body) == "" {
			return fmt.Errorf("%s 没有可执行的回滚语句", migration.DownPath)
		}
		if _, err := db.Exec(body); err != nil {
			return fmt.Errorf("%s 执行失败: %w", filepath.Base(migration.DownPath), err)
		}
		if _, err := db.Exec("DELETE FROM schema_migrations WHERE version = ?", version); err != nil {
			return fmt.Errorf("删除迁移版本 %s 失败: %w", version, err)
		}
		fmt.Printf("rolled back %s\n", version)
		return nil
	}
	return fmt.Errorf("迁移账本中的版本 %q 在当前迁移目录中不存在", version)
}

func baselineMigrations(db *sql.DB, files []migrationFile, requested string) error {
	target, err := strconv.Atoi(requested)
	if err != nil || target < 1 {
		return fmt.Errorf("版本必须是正整数，例如 015")
	}
	found := false
	for _, migration := range files {
		if migration.Version == target {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("迁移目录中不存在版本 %03d", target)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("迁移账本非空，拒绝覆盖已有历史")
	}
	for _, migration := range files {
		if migration.Version > target {
			break
		}
		if _, err := db.Exec("INSERT INTO schema_migrations(version) VALUES (?)", migration.Name); err != nil {
			return fmt.Errorf("记录 %s 失败: %w", migration.Name, err)
		}
	}
	fmt.Printf("baseline set to %03d; no SQL was executed\n", target)
	return nil
}

// loadMigration 读取迁移文件并剔除 sql-migrate 标记。
// up 方向遇到 `-- +migrate Down` 即截断，避免连带执行回滚语句。
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

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "错误: "+format+"\n", args...)
	os.Exit(1)
}
