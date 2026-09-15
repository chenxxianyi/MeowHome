// Package token 实现访问令牌的签发与校验。
//
// 令牌格式：base64url(JSON claims) + "." + hex(HMAC-SHA256(secret, base64url 部分))
// 校验同时覆盖签名与过期时间——只验签名不验 exp 会让令牌永不过期。
package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// 校验失败原因。
var (
	ErrMalformed = errors.New("token: malformed")
	ErrSignature = errors.New("token: invalid signature")
	ErrExpired   = errors.New("token: expired")
)

// Claims 访问令牌载荷。
type Claims struct {
	UserID    string `json:"uid"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// Sign 签发访问令牌。
func Sign(secret []byte, userID string, ttl time.Duration, now time.Time) (string, error) {
	c := Claims{
		UserID:    userID,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
	}
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(payload)
	return body + "." + sign(secret, body), nil
}

// Parse 校验并解析访问令牌。签名不匹配、格式错误、已过期都会返回错误。
func Parse(secret []byte, raw string, now time.Time) (*Claims, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, ErrMalformed
	}
	body, sig := parts[0], parts[1]

	// 常量时间比较，避免时序侧信道
	if !hmac.Equal([]byte(sig), []byte(sign(secret, body))) {
		return nil, ErrSignature
	}

	payload, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return nil, ErrMalformed
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, ErrMalformed
	}
	if c.UserID == "" {
		return nil, ErrMalformed
	}
	if c.ExpiresAt <= now.Unix() {
		return nil, ErrExpired
	}
	return &c, nil
}

func sign(secret []byte, body string) string {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(body))
	return hex.EncodeToString(m.Sum(nil))
}
