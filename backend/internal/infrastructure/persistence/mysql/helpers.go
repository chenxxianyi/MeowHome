// Package mysql 提供仓储通用辅助函数。
package mysql

import (
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// isDuplicate 判断是否为唯一键冲突错误。
// 必须容忍 err == nil：调用方普遍写成 `if isDuplicate(err) { ... }`，
// 成功路径同样会传进来，此前会走到 err.Error() 触发空指针 panic。
func isDuplicate(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		// 1062: duplicate entry
		return mysqlErr.Number == 1062
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate")
}
