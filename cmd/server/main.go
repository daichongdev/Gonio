package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"gonio/internal/config"
	"gonio/internal/database"
	"gonio/internal/middleware"
	"gonio/internal/mq"
	"gonio/internal/pkg/i18n"
	"gonio/internal/pkg/logger"
	"gonio/internal/pkg/validator"
	"gonio/internal/router"
	"gonio/internal/svc"
	"gonio/migration"

	"github.com/gin-gonic/gin"
)

func main() {
	if err := run(); err != nil {
		logger.Log.Errorw("server exited with error", "error", err)
	}
	logger.Log.Info("server exited")
}

func run() error {
	// 1. 加载配置（logger.Log 已有 init() 提供的 fallback，不会 nil panic）
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 2. 初始化日志
	logger.Init(&cfg.Log)
	defer logger.Sync()

	// 3. 设置 Gin 模式
	gin.SetMode(cfg.Server.Mode)

	// 4. 初始化 MySQL
	db, err := database.InitMySQL(&cfg.MySQL, &cfg.Log)
	if err != nil {
		return fmt.Errorf("init mysql: %w", err)
	}
	defer database.CloseMySQL(db)

	// 5. 初始化 Redis
	rdb, err := database.InitRedis(&cfg.Redis, &cfg.Log)
	if err != nil {
		return fmt.Errorf("init redis: %w", err)
	}
	defer database.CloseRedis(rdb)

	// 6. 数据库迁移（可配置）
	if cfg.Server.AutoMigrate {
		if err := migration.AutoMigrate(db); err != nil {
			return fmt.Errorf("auto migrate: %w", err)
		}
	}

	// 7. 初始化 JWT
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWT.Secret)

	// 8. 初始化验证器中文翻译
	validator.Init()

	// 9. 初始化多语言翻译
	i18n.Init()

	// 10. 初始化 MQ Publisher
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}
	mqPublisher, err := mq.NewPublisher(&cfg.MQ, rdb, sqlDB)
	if err != nil {
		return fmt.Errorf("init mq publisher: %w", err)
	}
	defer mqPublisher.Close()

	// 11. 初始化 ServiceContext（一行完成所有依赖接线）
	svcCtx := svc.NewServiceContext(cfg, db, rdb, mqPublisher)

	// 12. 路由（只传 ServiceContext）
	r := router.Setup(svcCtx, authMiddleware)

	// 13. 初始化 MQ Router
	mqRouter, err := mq.NewRouter(&cfg.MQ, rdb, sqlDB)
	if err != nil {
		return fmt.Errorf("init mq router: %w", err)
	}

	app := NewApp(cfg, r, mqRouter, rdb)

	runCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(runCtx); err != nil {
		return fmt.Errorf("run app: %w", err)
	}
	return nil
}
