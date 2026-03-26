package main

import (
	"context"
	"os/signal"
	"syscall"

	"goflow/internal/config"
	"goflow/internal/database"
	"goflow/internal/middleware"
	"goflow/internal/mq"
	"goflow/internal/pkg/i18n"
	"goflow/internal/pkg/logger"
	"goflow/internal/pkg/validator"
	"goflow/internal/router"
	"goflow/internal/svc"
	"goflow/migration"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 加载配置（logger.Log 已有 init() 提供的 fallback，不会 nil panic）
	cfg, err := config.Load()
	if err != nil {
		logger.Log.Fatalf("failed to load config: %v", err)
	}

	// 2. 初始化日志
	logger.Init(&cfg.Log)
	defer logger.Sync()

	// 3. 设置 Gin 模式
	gin.SetMode(cfg.Server.Mode)

	// 4. 初始化 MySQL
	db, err := database.InitMySQL(&cfg.MySQL, &cfg.Log)
	if err != nil {
		logger.Log.Fatalf("init mysql failed: %v", err)
	}
	defer database.CloseMySQL(db)

	// 5. 初始化 Redis
	rdb, err := database.InitRedis(&cfg.Redis, &cfg.Log)
	if err != nil {
		logger.Log.Fatalf("init redis failed: %v", err)
	}
	defer database.CloseRedis(rdb)

	// 6. 数据库迁移（可配置）
	if cfg.Server.AutoMigrate {
		if err := migration.AutoMigrate(db); err != nil {
			logger.Log.Fatalf("auto migrate failed: %v", err)
		}
	}

	// 7. 初始化 JWT
	if cfg.JWT.Secret == "" {
		logger.Log.Fatal("jwt secret is empty")
	}
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWT.Secret)

	// 8. 初始化验证器中文翻译
	validator.Init()

	// 9. 初始化多语言翻译
	i18n.Init()

	// 10. 初始化 MQ Publisher
	sqlDB, err := db.DB()
	if err != nil {
		logger.Log.Fatalf("get sql.DB failed: %v", err)
	}
	mqPublisher, err := mq.NewPublisher(&cfg.MQ, rdb, sqlDB)
	if err != nil {
		logger.Log.Fatalf("init mq publisher failed: %v", err)
	}
	defer mqPublisher.Close()

	// 11. 初始化 ServiceContext（一行完成所有依赖接线）
	svcCtx := svc.NewServiceContext(cfg, db, rdb, mqPublisher)

	// 12. 路由（只传 ServiceContext）
	r := router.Setup(svcCtx, authMiddleware)

	// 13. 初始化 MQ Router
	mqRouter, err := mq.NewRouter(&cfg.MQ, rdb, sqlDB)
	if err != nil {
		logger.Log.Fatalf("init mq router failed: %v", err)
	}

	app := NewApp(cfg, r, mqRouter, rdb)

	runCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(runCtx); err != nil {
		logger.Log.Fatalf("app run failed: %v", err)
	}

	logger.Log.Info("server exited")
}
