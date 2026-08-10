package app

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

func timeNowFunc() time.Time { return time.Now() }

// randomToken 生成随机 token 字符串（raw refresh token）。
func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", timeNowUnix())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func timeNowUnix() int64 {
	return timeNowFunc().Unix()
}

// hmacSign 使用密钥对数据做 HMAC-SHA256 签名。
func hmacSign(key, data []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(data)
	return m.Sum(nil)
}

// structToString 将任意结构序列化为字符串（用于签名载荷）。
func structToString(v any) string {
	return fmt.Sprintf("%v", v)
}

// HashPassword 简化版密码哈希（B3：SHA256 + 随机盐；B4 起替换为 bcrypt）。
func HashPassword(password string) string {
	salt := randomSalt()
	h := sha256.New()
	h.Write([]byte(salt + password))
	sum := hex.EncodeToString(h.Sum(nil))
	return salt + ":" + sum
}

// checkPassword 校验密码。
func checkPassword(hashed, password string) bool {
	// 格式：盐:哈希
	parts := splitFirst(hashed, ':')
	if len(parts) != 2 {
		return false
	}
	h := sha256.New()
	h.Write([]byte(parts[0] + password))
	return hex.EncodeToString(h.Sum(nil)) == parts[1]
}

func splitFirst(s string, sep byte) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func randomSalt() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
