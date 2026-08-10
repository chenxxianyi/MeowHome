// Package clock 定义可注入时钟，领域层不直接依赖 time.Now。
package clock

import "time"

// Clock 可注入时钟接口。
type Clock interface {
	Now() time.Time
}

// SystemClock 返回当前时间。
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

// FixedClock 用于测试的固定时钟。
type FixedClock struct{ T time.Time }

func (f FixedClock) Now() time.Time { return f.T }
