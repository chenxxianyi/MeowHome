// Package id 定义 ID 生成器。
package id

import (
	"crypto/rand"
	"io"
	"time"

	"github.com/oklog/ulid/v2"
)

// IDGenerator 生成业务主键 ULID。
type IDGenerator interface {
	New() string
}

// ULIDGenerator 基于 ULID 的实现。
type ULIDGenerator struct{}

// monotonicReader 实现 ulid.MonotonicReader（io.Reader + MonotonicRead）。
// 底层从 crypto/rand 读取随机字节，ulid 库的 LockedMonotonicReader
// 负责在同一毫秒内保证单调递增（溢出时进位）。
type monotonicReader struct{}

// Read 实现 io.Reader：从 crypto/rand 读取随机字节。
func (monotonicReader) Read(p []byte) (int, error) {
	return rand.Read(p)
}

// MonotonicRead 实现 ulid.MonotonicReader：
// 在给定毫秒 ms 下填充 buf，保证单调递增（由 LockedMonotonicReader 保障）。
func (monotonicReader) MonotonicRead(ms uint64, buf []byte) error {
	_, err := rand.Read(buf)
	return err
}

// entropy 用 LockedMonotonicReader 包裹，保证并发安全且单调递增。
var entropy = &ulid.LockedMonotonicReader{MonotonicReader: monotonicReader{}}

// New 生成一个按时间排序、线程安全、单调递增的 ULID。
func (ULIDGenerator) New() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}

// _ 确保 io 被使用（Read 方法）。
var _ io.Reader = monotonicReader{}
