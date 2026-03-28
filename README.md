# GoFlow - Golang Gin Backend Scaffold with Redis Rate Limiter, JWT Auth, and Clean Architecture

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/)
[![Gin](https://img.shields.io/badge/Gin-Web%20Framework-00A86B)](https://github.com/gin-gonic/gin)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)

GoFlow 是一个面向生产环境的 **Golang 后端脚手架**，基于 **Gin + GORM + Redis + MySQL**，内置 **IP + 路由限流（Redis Lua）**、JWT 认证、结构化日志、消息队列、多语言校验、优雅停机。

If you are looking for a **Go API starter template**, **Gin Redis rate limiter example**, or a **clean architecture backend scaffold**, GoFlow is built for that.

## Why GoFlow

- 面向高并发 API 服务：连接池、超时、重试、限流、日志链路完整
- 清晰分层架构：`Handler -> Service -> Repository`
- 中间件完善：RequestID、Logger、Recovery、CORS、I18n、Auth、Rate Limit
- 可扩展性强：通过 `ServiceContext` 统一依赖注入
- 可观测性友好：Zap 结构化日志 + DB/Redis 日志钩子

## Core Features

- **Gin Web API Framework**
- **GORM + MySQL** data access layer
- **Redis cache and Redis-based rate limiter**
- **IP + Route Rate Limit** via Redis Lua script (atomic)
- **JWT Authentication** for app/admin
- **Watermill MQ** with Redis Streams / MySQL backend
- **I18n Validation & Error Messages**
- **Graceful Shutdown**

## Architecture

```text
.
├── cmd/
│   └── server/         # 服务启动入口
├── config/             # 配置文件
├── internal/
│   ├── config/
│   ├── database/
│   ├── handler/
│   ├── middleware/
│   ├── model/
│   ├── mq/
│   ├── pkg/            # errcode / response / validator / ratelimit
│   ├── repository/
│   ├── router/
│   ├── service/
│   └── svc/            # ServiceContext 依赖注入
├── migration/
├── go.mod
└── Makefile
```

## Quick Start

### Requirements

- Go 1.25+
- MySQL 8.0+
- Redis 6.0+

### Run

```bash
git clone https://github.com/your-username/goflow.git
cd goflow

go mod tidy
make run
```

Health check:

```bash
curl http://localhost:8080/health
```

## Rate Limiter Example

GoFlow supports Redis-based API rate limiting by **IP + route + method**.

- Product list API: `1 request / 1 second`
- Product create API: `1 request / 3 seconds`

关键词（SEO）：`Golang 限流器`、`Gin 限流中间件`、`Redis Lua 限流`、`IP 路由限流`、`Go API Rate Limiter`

## Typical Use Cases

- 电商 API / 用户中心 / 管理后台
- 需要登录认证 + 限流 + 日志审计的业务系统
- 需要快速落地的 Go 微服务或单体 API 项目

## API Overview

- App APIs: `/app/v1/*`
- Admin APIs: `/admin/v1/*`
- Health Check: `/health`

## Roadmap

- Sliding window / token bucket rate limit strategy
- OpenAPI/Swagger docs
- Prometheus metrics and tracing integration

## SEO Keywords

Golang backend scaffold, Gin boilerplate, Go web api template, Redis rate limiter, Gin rate limit middleware, JWT auth in Go, GORM MySQL starter, Clean Architecture Go.

## Contributing

Issues and PRs are welcome.

If this project helps you, please consider giving it a ⭐ on GitHub.

## License

MIT License. See [LICENSE](./LICENSE).
