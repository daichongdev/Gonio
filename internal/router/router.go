package router

import (
	"context"
	"net/http"
	"time"

	"goflow/internal/handler"
	"goflow/internal/middleware"
	"goflow/internal/pkg/response"
	"goflow/internal/router/admin"
	"goflow/internal/router/app"
	"goflow/internal/svc"

	"github.com/gin-gonic/gin"
)

// Setup 接收 ServiceContext，内部创建所有 handler 并注册路由。
// 新增模块只需在此处添加 handler 和路由，无需修改函数签名。
func Setup(svcCtx *svc.ServiceContext, auth *middleware.AuthMiddleware) *gin.Engine {
	r := gin.New()

	// 全局中间件
	r.Use(middleware.RequestID())
	r.Use(middleware.I18n())
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS(svcCtx.Config.Server.CORSOrigins))

	// 健康检查：检测 MySQL 和 Redis 的真实连通性
	r.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		if err := svcCtx.HealthCheck(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, response.Response{
				Code:    -1,
				Message: "unhealthy: " + err.Error(),
			})
			return
		}
		response.Success(c, gin.H{"status": "ok"})
	})

	// 创建 handler（从 ServiceContext 获取依赖）
	productHandler := handler.NewProductHandler(svcCtx.ProductSvc, svcCtx.MQPublisher)
	userHandler := handler.NewUserHandler(svcCtx.UserSvc)
	adminHandler := handler.NewAdminHandler(svcCtx.AdminSvc)

	// App 客户端接口
	appGroup := r.Group("/app/v1")
	app.RegisterUserRoutes(appGroup, userHandler)
	app.RegisterProductRoutes(appGroup, productHandler)

	// App 需要认证的接口
	appAuth := appGroup.Group("")
	appAuth.Use(auth.AppAuth())
	// 后续扩展 App 需认证的接口

	// 管理后台接口
	adminGroup := r.Group("/admin/v1")
	admin.RegisterAuthRoutes(adminGroup, adminHandler)

	// 管理后台全部需认证
	adminAuth := adminGroup.Group("")
	adminAuth.Use(auth.AdminAuth())
	admin.RegisterProductRoutes(adminAuth, productHandler)

	return r
}
