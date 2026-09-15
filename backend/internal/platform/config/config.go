// Package config 提供统一的配置加载。
// 配置来自环境变量或 .env 文件；生产环境禁止提交真实密钥。
package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 服务全部配置。
//
// 注意：字段必须带 mapstructure tag。mapstructure 的默认字段匹配只忽略大小写、
// 不忽略下划线，因此 `jwt_secret` 无法匹配到 `JWTSecret`，会静默变成零值。
type Config struct {
	App     App     `mapstructure:"app"`
	Server  Server  `mapstructure:"server"`
	MySQL   MySQL   `mapstructure:"mysql"`
	Auth    Auth    `mapstructure:"auth"`
	Storage Storage `mapstructure:"storage"`
	AI      AI      `mapstructure:"ai"`
	OCR     OCR     `mapstructure:"ocr"`
	Redis   Redis   `mapstructure:"redis"`
	Rate    Rate    `mapstructure:"rate"`
}

// App 应用级配置。
type App struct {
	Env            string   `mapstructure:"env"`
	Name           string   `mapstructure:"name"`
	LogLevel       string   `mapstructure:"log_level"`
	LogFormat      string   `mapstructure:"log_format"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// Server HTTP 服务配置。
type Server struct {
	Port            string        `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	MaxUploadSizeMB int64         `mapstructure:"max_upload_size_mb"`
}

// MySQL 数据库配置。
type MySQL struct {
	Host            string        `mapstructure:"host"`
	Port            string        `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	TLS             bool          `mapstructure:"tls"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// Auth 认证配置。
type Auth struct {
	JWTSecret  string        `mapstructure:"jwt_secret"`
	AccessTTL  time.Duration `mapstructure:"access_ttl"`
	RefreshTTL time.Duration `mapstructure:"refresh_ttl"`
	BCryptCost int           `mapstructure:"bcrypt_cost"`
}

// Storage 文件存储配置。
type Storage struct {
	Driver   string `mapstructure:"driver"`
	LocalDir string `mapstructure:"local_dir"`
	BaseURL  string `mapstructure:"base_url"`
}

// AI OpenAI-compatible Provider 配置。
type AI struct {
	Enabled    bool          `mapstructure:"enabled"`
	BaseURL    string        `mapstructure:"base_url"`
	APIKey     string        `mapstructure:"api_key"`
	Model      string        `mapstructure:"model"`
	Timeout    time.Duration `mapstructure:"timeout"`
	MaxRetries int           `mapstructure:"max_retries"`
}

// OCR Provider 配置。
type OCR struct {
	Enabled bool          `mapstructure:"enabled"`
	BaseURL string        `mapstructure:"base_url"`
	APIKey  string        `mapstructure:"api_key"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// Redis 预留配置（MVP 阶段可缺省）。
type Redis struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// Rate 基础限流配置。
type Rate struct {
	PerMinute int `mapstructure:"per_minute"`
}

// envFile 默认配置文件路径（相对工作目录）。
const envFile = ".env"

// Load 从 .env 文件与环境变量加载配置。
// 优先级：进程真实环境变量 > .env 文件 > 内置默认值。
func Load() (*Config, error) {
	// 先把 .env 注入进程环境，AutomaticEnv/BindEnv 才能看到它。
	loadDotEnv(envFile)

	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults(v)
	bindEnvs(v)

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	return cfg, cfg.Validate()
}

// loadDotEnv 把 .env 中的键值注入进程环境。
// 已存在的真实环境变量优先，不会被覆盖（标准 dotenv 语义）。
// 文件不存在时静默跳过，全部走内置默认值。
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])

		// 去引号；未加引号时剥离行尾注释（" # ..."）
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
			val = val[1 : len(val)-1]
		} else if i := strings.Index(val, " #"); i >= 0 {
			val = strings.TrimSpace(val[:i])
		}

		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue // 真实环境变量优先
		}
		_ = os.Setenv(key, val)
	}
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

// bindEnvs 把配置键显式绑定到 .env 中的变量名。
//
// 只有「.env 变量名 ≠ 配置键大写化」时才需要显式绑定，例如
// server.port 默认推导 SERVER_PORT，但 .env 用的是 APP_PORT。
// 其余键（如 mysql.host → MYSQL_HOST）由 AutomaticEnv + 键名替换自动覆盖。
func bindEnvs(v *viper.Viper) {
	pairs := map[string]string{
		"app.env":             "APP_ENV",
		"app.log_level":       "LOG_LEVEL",
		"app.log_format":      "LOG_FORMAT",
		"app.allowed_origins": "ALLOWED_ORIGINS",

		"server.port":               "APP_PORT",
		"server.read_timeout":       "READ_TIMEOUT",
		"server.write_timeout":      "WRITE_TIMEOUT",
		"server.shutdown_timeout":   "SHUTDOWN_TIMEOUT",
		"server.max_upload_size_mb": "MAX_UPLOAD_SIZE_MB",

		"auth.jwt_secret":  "JWT_SECRET",
		"auth.access_ttl":  "JWT_ACCESS_TTL",
		"auth.refresh_ttl": "JWT_REFRESH_TTL",
		"auth.bcrypt_cost": "BCRYPT_COST",

		"rate.per_minute": "RATE_LIMIT_PER_MINUTE",
	}
	for key, env := range pairs {
		_ = v.BindEnv(key, env)
	}
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
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("config: auth.jwt_secret is required")
	}
	return nil
}

func isPlaceholderSecret(s string) bool {
	return strings.Contains(strings.ToLower(s), "replace") || s == "changeme"
}
