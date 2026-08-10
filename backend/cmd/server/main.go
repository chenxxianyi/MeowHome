// Package main 启动 MeowHome 后端服务。
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/app"
	"github.com/meowhome/backend/internal/infrastructure/persistence/mysql"
	"github.com/meowhome/backend/internal/platform/config"
	"github.com/meowhome/backend/internal/transport/http/handler"
	"github.com/meowhome/backend/internal/transport/http/router"
)

func main() {
	logger := newLogger()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("config", zap.Error(err))
	}

	logger.Info("starting meowhome backend",
		zap.String("env", cfg.App.Env),
		zap.String("port", cfg.Server.Port),
	)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 连接 MySQL（B3 阶段接入；连接失败时降级运行，健康检查将返回 down）
	db := connectDB(cfg, logger)

	// 组装仓储层
	userRepo := mysql.NewUserRepo(db)
	familyRepo := mysql.NewFamilyRepo(db)
	memberRepo := mysql.NewMemberRepo(db)
	catRepo := mysql.NewCatRepo(db)
	healthRepo := mysql.NewHealthProfileRepo(db)
	mediaRepo := mysql.NewMediaRepo(db)
	refreshRepo := mysql.NewRefreshTokenRepo(db)
	auditRepo := mysql.NewAuditRepo(db)
	_ = mediaRepo

	// 组装应用层
	authSvc := app.NewAuthService(userRepo, refreshRepo, []byte(cfg.Auth.JWTSecret), cfg.Auth.AccessTTL)
	familySvc := app.NewFamilyService(familyRepo, memberRepo, userRepo, auditRepo)
	catSvc := app.NewCatService(catRepo, healthRepo, memberRepo, auditRepo, familyRepo)
	memberSvc := app.NewMemberService(memberRepo, userRepo, familyRepo)

	// 组装 HTTP 层
	hdl := handler.New(authSvc, familySvc, catSvc, memberSvc)
	health := &appHealth{db: db}
	srv.Handler = router.New(cfg, logger, nil, health, hdl)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("shutdown", zap.Error(err))
	}
	logger.Info("server stopped")
}

// connectDB 建立 MySQL 连接；失败时返回 nil 并降级。
func connectDB(cfg *config.Config, logger *zap.Logger) *gorm.DB {
	db, err := mysql.New(cfg.MySQL)
	if err != nil {
		logger.Warn("mysql connect failed (degraded mode)", zap.Error(err))
		return nil
	}
	return db
}

func newLogger() *zap.Logger {
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "ts"
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncodeLevel = zapcore.CapitalLevelEncoder
	cfg.EncodeDuration = zapcore.SecondsDurationEncoder

	if os.Getenv("APP_ENV") == "development" {
		return zap.New(zapcore.NewCore(
			zapcore.NewConsoleEncoder(cfg),
			zapcore.Lock(os.Stdout),
			zapcore.DebugLevel,
		))
	}
	return zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		zapcore.Lock(os.Stdout),
		zapcore.InfoLevel,
	))
}

// appHealth 结合 DB 的实际健康检查。
type appHealth struct {
	db *gorm.DB
}

func (h *appHealth) Liveness() (any, error) {
	return gin.H{"status": "up", "version": "0.2.0"}, nil
}

func (h *appHealth) Readiness() (any, error) {
	if h.db != nil {
		if err := mysql.Healthy(h.db); err != nil {
			return nil, err
		}
	}
	return gin.H{"status": "up"}, nil
}
