// Package mysql 提供仓储通用辅助函数。
package mysql

import (
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// isDuplicate 判断是否为唯一键冲突错误。
func isDuplicate(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		// 1062: duplicate entry
		return mysqlErr.Number == 1062
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate")
}
