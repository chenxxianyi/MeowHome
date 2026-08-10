// Package config 提供统一的配置加载。
// 配置来自环境变量或 .env 文件；生产环境禁止提交真实密钥。
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 服务全部配置。
type Config struct {
	App     App
	Server  Server
	MySQL   MySQL
	Auth    Auth
	Storage Storage
	AI      AI
	OCR     OCR
	Redis   Redis
	Rate    Rate
}

// App 应用级配置。
type App struct {
	Env            string
	Name           string
	LogLevel       string
	LogFormat      string
	AllowedOrigins []string
}

// Server HTTP 服务配置。
type Server struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	MaxUploadSizeMB int64
}

// MySQL 数据库配置。
type MySQL struct {
	Host            string
	Port            string
	User            string
	Password        string
	Database        string
	TLS             bool
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Auth 认证配置。
type Auth struct {
	JWTSecret  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	BCryptCost int
}

// Storage 文件存储配置。
type Storage struct {
	Driver   string
	LocalDir string
	BaseURL  string
}

// AI OpenAI-compatible Provider 配置。
type AI struct {
	Enabled    bool
	BaseURL    string
	APIKey     string
	Model      string
	Timeout    time.Duration
	MaxRetries int
}

// OCR Provider 配置。
type OCR struct {
	Enabled bool
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// Redis 预留配置（MVP 阶段可缺省）。
type Redis struct {
	Addr     string
	Password string
	DB       int
}

// Rate 基础限流配置。
type Rate struct {
	PerMinute int
}

// Load 从环境变量加载配置。
func Load() (*Config, error) {
	v := viper.New()

	// 显式绑定 .env 文件
	v.SetConfigFile(".env")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 允许忽略 .env 不存在的错误
	_ = v.ReadInConfig()

	setDefaults(v)
	bindEnvs(v)

	if err := v.Unmarshal(&Config{}); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	return cfg, cfg.Validate()
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.env", "development")
	v.SetDefault("app.name", "meowhome-backend")
	v.SetDefault("app.log_level", "info")
	v.SetDefault("app.log_format", "console")
	v.SetDefault("app.allowed_origins", "http://localhost:5173")

	v.SetDefault("server.port", "8080")
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.shutdown_timeout", "15s")
	v.SetDefault("server.max_upload_size_mb", 20)

	v.SetDefault("mysql.host", "127.0.0.1")
	v.SetDefault("mysql.port", "3306")
	v.SetDefault("mysql.user", "meowhome")
	v.SetDefault("mysql.password", "changeme")
	v.SetDefault("mysql.database", "meowhome")
	v.SetDefault("mysql.tls", false)
	v.SetDefault("mysql.max_open_conns", 50)
	v.SetDefault("mysql.max_idle_conns", 10)
	v.SetDefault("mysql.conn_max_lifetime", "1h")

	v.SetDefault("auth.jwt_secret", "replace-with-strong-secret")
	v.SetDefault("auth.access_ttl", "15m")
	v.SetDefault("auth.refresh_ttl", "720h")
	v.SetDefault("auth.bcrypt_cost", 12)

	v.SetDefault("storage.driver", "local")
	v.SetDefault("storage.local_dir", "./storage")
	v.SetDefault("storage.base_url", "http://localhost:8080/static")

	v.SetDefault("ai.enabled", false)
	v.SetDefault("ai.timeout", "30s")
	v.SetDefault("ai.max_retries", 2)

	v.SetDefault("ocr.enabled", false)
	v.SetDefault("ocr.timeout", "30s")

	v.SetDefault("redis.db", 0)

	v.SetDefault("rate.per_minute", 120)
}

func bindEnvs(v *viper.Viper) {
	_ = v.BindEnv("app.env")
	_ = v.BindEnv("server.port", "APP_PORT")
	// 其余环境变量名与 key 一致（大写化），AutomaticEnv + masterkey replacer 已覆盖
}

// Validate 校验生产必需配置。
func (c *Config) Validate() error {
	if c.App.Env == "production" {
		if c.Auth.JWTSecret == "" || isPlaceholderSecret(c.Auth.JWTSecret) {
			return fmt.Errorf("config: production requires a real JWT secret")
		}
		if c.MySQL.User == "" || c.MySQL.Host == "" {
			return fmt.Errorf("config: production requires MySQL configuration")
		}
	}
	if c.MySQL.Database == "" {
		return fmt.Errorf("config: mysql.database is required")
	}
	return nil
}

func isPlaceholderSecret(s string) bool {
	return strings.Contains(strings.ToLower(s), "replace") || s == "changeme"
}
