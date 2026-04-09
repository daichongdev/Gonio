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

### Add Config
```
/config/config-dev.yaml
server:
  port: 8080
  mode: release # debug / release
  read_timeout: 10
  write_timeout: 10
  auto_migrate: true

mysql:
  host: 127.0.0.1
  port: 3306
  username: root
  password: "123456"
  database: silk_route
  max_idle_conns: 10
  max_open_conns: 100
  max_lifetime: 3600        # 连接最大存活时间(秒)
  conn_max_idle_time: 600   # 空闲连接最大存活时间(秒)
  dial_timeout: 5           # 连接超时(秒)
  read_timeout: 10          # 读超时(秒)
  write_timeout: 10         # 写超时(秒)
  ping_timeout: 3           # 探活超时(秒)
  prepare_stmt: true        # 缓存预编译语句，减少 SQL 解析开销
  skip_default_transaction: true  # 非事务查询不包裹事务，提升约 30% 性能

mq:
  driver: redis          # redis 或 mysql
  consumer_group: goflow-group
  topic_concurrency:     # 按 topic 单独配置，未配置默认 1
    email: 3
    sms: 1
    stats: 2
  default_max_len: 20   # stream 全局默认最大长度
  topic_max_len:           # 按 topic 单独设置，优先级高于全局
    email: 10
    sms: 2000
    stats: 10000
  trim_interval: 3600      # 定期 XTRIM 间隔（秒），0 为不启用

redis:
  addr: 127.0.0.1:6379
  password: ""
  db: 2
  pool_size: 100
  min_idle_conns: 10        # 最小空闲连接数
  max_idle_conns: 50        # 最大空闲连接数
  pool_timeout: 5           # 获取连接池超时(秒)
  dial_timeout: 5           # 连接超时(秒)
  read_timeout: 3           # 读超时(秒)
  write_timeout: 3          # 写超时(秒)
  conn_max_idle_time: 300   # 空闲连接最大存活时间(秒)
  conn_max_lifetime: 3600   # 连接最大存活时间(秒)
  ping_timeout: 3           # 探活超时(秒)
  max_retries: 3            # 命令失败最大重试次数
  min_retry_backoff: 8      # 最小重试退避时间(毫秒)
  max_retry_backoff: 512    # 最大重试退避时间(毫秒)

jwt:
  secret: "goflow-secret-key-change-me"
  expire: 604800 # seconds

log:
  mode: dev # dev / prod
  level: info
  sql_level: warn
  file_path: logs/app.log  # 日志文件路径，空则只输出到控制台
  max_size: 100            # 单文件最大 MB
  max_backups: 7           # 保留旧文件数
  max_age: 30              # 保留天数
  compress: true           # 压缩归档

smtp:
  host: smtp.example.com
  port: 465
  username: no-reply@example.com
  password: ""
  from: no-reply@example.com
  tls: true

```

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
