// Package main 提供数据库迁移执行器。
//
// 用法（在 backend/ 目录下执行，配置取自 .env）：
//
//	go run ./cmd/migrate           # 按文件名顺序执行全部 *.up.sql
//	go run ./cmd/migrate -down     # 逆序执行全部 *.down.sql
//	go run ./cmd/migrate -dir migrations -dry
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
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"github.com/meowhome/backend/internal/platform/config"
)

func main() {
	down := flag.Bool("down", false, "执行 down 迁移（逆序）")
	dir := flag.String("dir", "migrations", "迁移文件目录")
	dry := flag.Bool("dry", false, "只打印将执行的语句，不落库")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fatal("加载配置失败: %v", err)
	}

	pattern := "*.up.sql"
	if *down {
		pattern = "*.down.sql"
	}
	files, err := filepath.Glob(filepath.Join(*dir, pattern))
	if err != nil {
		fatal("扫描迁移目录失败: %v", err)
	}
	if len(files) == 0 {
		fatal("在 %s 下未找到 %s", *dir, pattern)
	}
	sort.Strings(files)
	if *down {
		for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
			files[i], files[j] = files[j], files[i]
		}
	}

	var db *sql.DB
	if !*dry {
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
	}

	for _, f := range files {
		base := filepath.Base(f)
		body, err := loadMigration(f, *down)
		if err != nil {
			fatal("%s: 读取失败: %v", base, err)
		}
		if strings.TrimSpace(body) == "" {
			fmt.Printf("skip    %s（无有效语句）\n", base)
			continue
		}
		if *dry {
			fmt.Printf("dry-run %s\n%s\n", base, body)
			continue
		}
		if _, err := db.Exec(body); err != nil {
			fatal("%s: 执行失败: %v", base, err)
		}
		fmt.Printf("applied %s\n", base)
	}

	if !*dry {
		fmt.Println("迁移完成")
	}
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
