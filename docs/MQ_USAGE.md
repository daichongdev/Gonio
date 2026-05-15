# Gonio 消息队列使用文档

## 目录

1. [架构概述](#架构概述)
2. [快速开始](#快速开始)
3. [配置说明](#配置说明)
4. [注册新消息类型](#注册新消息类型)
5. [发布消息](#发布消息)
6. [消费消息](#消费消息)
7. [高级特性](#高级特性)
8. [最佳实践](#最佳实践)
9. [故障排查](#故障排查)

---

## 架构概述

Gonio 消息队列基于 [Watermill](https://github.com/ThreeDotsLabs/watermill) 构建，支持 **Redis Streams** 和 **MySQL** 两种后端存储。

### 核心组件

```
┌─────────────────────────────────────────────────────────────┐
│                      应用层 (Service)                        │
│  调用 publisher.PublishEmail() / PublishSMS() 等方法         │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                   Publisher (发布者)                         │
│  序列化 payload → 发送到 topic                               │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              消息队列后端 (Redis / MySQL)                    │
│  存储消息，支持持久化、重试、死信队列                         │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                   Router (消费者路由)                        │
│  从 topic 拉取消息 → 反序列化 → 调用 handler                 │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                   Handler (业务处理)                         │
│  handleEmail / handleSMS / handleStats 等                   │
└─────────────────────────────────────────────────────────────┘
```

### 关键特性

- ✅ **统一注册表**：新增消息类型只需修改一个文件
- ✅ **类型安全**：使用泛型确保编译时类型检查
- ✅ **自动重试**：失败消息自动重试 3 次
- ✅ **死信队列**：超过重试次数的消息进入 `mq.poison` 队列
- ✅ **并发消费**：每个 topic 可配置独立的消费者数量
- ✅ **消息裁剪**：Redis 模式支持定期 XTRIM，防止内存溢出
- ✅ **链路追踪**：消息 UUID 复用 HTTP 请求的 `request_id`

---

## 快速开始

### 1. 配置消息队列

编辑 `config/config.yaml`：

```yaml
mq:
  driver: redis              # 后端存储：redis 或 mysql
  consumer_group: gonio-group # Redis Streams 消费者组名
  
  # 并发消费配置（按 topic 短名称配置）
  topic_concurrency:
    email: 3                 # 邮件队列 3 个并发消费者
    sms: 1                   # 短信队列 1 个消费者
    stats: 2                 # 统计队列 2 个消费者
  
  # 消息队列长度限制（仅 Redis 生效）
  default_max_len: 20        # 全局默认最大长度
  topic_max_len:             # 按 topic 单独设置
    email: 10                # 邮件队列最多保留 10 条
    sms: 2000                # 短信队列最多保留 2000 条
    stats: 10000             # 统计队列最多保留 10000 条
  
  trim_interval: 3600        # 定期裁剪间隔（秒），0 为不启用
```

### 2. 启动服务

消息队列会在服务启动时自动初始化：

```go
// cmd/server/main.go
func main() {
    // ...
    
    // 创建发布者
    mqPublisher, err := mq.NewPublisher(&cfg.MQ, rdb, sqlDB)
    if err != nil {
        log.Fatal(err)
    }
    
    // 创建消费者路由
    mqRouter, err := mq.NewRouter(&cfg.MQ, rdb, sqlDB)
    if err != nil {
        log.Fatal(err)
    }
    
    // 启动消费者（在 goroutine 中运行）
    go func() {
        if err := mqRouter.Run(ctx); err != nil {
            log.Error("mq router stopped", err)
        }
    }()
    
    // 启动定期裁剪（仅 Redis 模式）
    mq.StartTrimmer(ctx, &cfg.MQ, rdb)
    
    // ...
}
```

### 3. 发布消息

在业务代码中调用 Publisher：

```go
// 示例：用户注册后发送欢迎邮件
func (s *UserService) Register(ctx context.Context, req RegisterRequest) error {
    // 1. 创建用户
    user := &model.User{
        Username: req.Username,
        Email:    req.Email,
    }
    if err := s.repo.Create(ctx, user); err != nil {
        return err
    }
    
    // 2. 发布邮件任务到消息队列（异步）
    err := s.publisher.PublishEmail(ctx, mq.EmailPayload{
        To:      user.Email,
        Subject: "欢迎注册 Gonio",
        Body:    "感谢您的注册！",
    })
    if err != nil {
        // 邮件发送失败不影响注册流程，记录日志即可
        logger.Log.Warnw("publish email failed", "err", err)
    }
    
    return nil
}
```

---

## 配置说明

### MQConfig 字段详解

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `driver` | string | 后端存储类型：`redis` 或 `mysql` | 必填 |
| `consumer_group` | string | Redis Streams 消费者组名（仅 Redis 模式） | 必填 |
| `topic_concurrency` | map[string]int | 按 topic 短名称配置并发消费者数 | 1 |
| `default_max_len` | int64 | Redis Stream 全局默认最大长度，0 为不限制 | 0 |
| `topic_max_len` | map[string]int | 按 topic 短名称单独设置最大长度 | - |
| `trim_interval` | int | 定期 XTRIM 间隔（秒），0 为不启用 | 0 |

### 配置示例

#### Redis 模式（推荐）

```yaml
mq:
  driver: redis
  consumer_group: gonio-group
  topic_concurrency:
    email: 3      # 邮件发送并发度高
    sms: 1        # 短信有频率限制，单线程
    stats: 5      # 统计数据量大，高并发
  default_max_len: 1000
  topic_max_len:
    email: 100    # 邮件队列较短，快速消费
    stats: 50000  # 统计队列较长，允许积压
  trim_interval: 600  # 每 10 分钟裁剪一次
```

#### MySQL 模式

```yaml
mq:
  driver: mysql
  consumer_group: ""  # MySQL 模式不需要消费者组
  topic_concurrency:
    email: 2
    sms: 1
    stats: 3
  # MySQL 模式不支持 max_len 和 trim_interval
```

---

## 注册新消息类型

### 步骤 1：在 `internal/mq/types.go` 中定义

```go
// 1. 定义 Topic 常量
const TopicAppPush = "silk_route.app_push"

// 2. 定义 Payload 结构体
type AppPushPayload struct {
    UserID  uint   `json:"user_id"`
    Title   string `json:"title"`
    Content string `json:"content"`
    Data    map[string]any `json:"data,omitempty"`
}

// 3. 实现 Handler 函数
func handleAppPush(msg *message.Message, payload AppPushPayload) error {
    // TODO: 调用推送服务（如 Firebase / 极光推送）
    logger.Log.Infow("[mq] send app push",
        "msg_uuid", msg.UUID,
        "user_id", payload.UserID,
        "title", payload.Title,
    )
    return nil
}

// 4. 在 init() 中注册（仅需添加这 4 行）
func init() {
    // ... 其他注册 ...
    
    Register(MessageType[AppPushPayload]{
        ShortName: "app_push",  // 配置文件中使用的短名称
        Topic:     TopicAppPush,
        Handler:   handleAppPush,
    })
}
```

### 步骤 2：更新配置文件

```yaml
mq:
  topic_concurrency:
    app_push: 2  # 新增：应用推送并发度
  topic_max_len:
    app_push: 5000  # 新增：队列最大长度
```

### 步骤 3：在 Publisher 中添加便捷方法（可选）

```go
// internal/mq/publisher.go

// PublishAppPush 发布应用推送任务
func (p *Publisher) PublishAppPush(ctx context.Context, payload AppPushPayload) error {
    return p.Publish(ctx, TopicAppPush, payload)
}
```

### 完成！

重启服务后，新消息类型自动生效：
- ✅ Router 自动注册 handler
- ✅ Trimmer 自动识别配置
- ✅ Publisher 支持发布

---

## 发布消息

### 方法 1：使用类型安全的便捷方法（推荐）

```go
// 发布邮件
err := publisher.PublishEmail(ctx, mq.EmailPayload{
    To:      "user@example.com",
    Subject: "订单确认",
    Body:    "您的订单已确认",
})

// 发布短信
err := publisher.PublishSMS(ctx, mq.SMSPayload{
    Phone:   "+86 138 0000 0000",
    Content: "验证码：123456",
})

// 发布统计事件
err := publisher.PublishStats(ctx, mq.StatsPayload{
    Event:  "user.login",
    UserID: 123,
    Properties: map[string]any{
        "ip":         "192.168.1.1",
        "user_agent": "Mozilla/5.0",
    },
})
```

### 方法 2：使用泛型方法

```go
// 定义消息类型
var emailType = mq.MessageType[mq.EmailPayload]{
    ShortName: "email",
    Topic:     mq.TopicEmail,
    Handler:   nil, // 发布时不需要 handler
}

// 发布
err := mq.PublishTyped(publisher, ctx, emailType, mq.EmailPayload{
    To:      "user@example.com",
    Subject: "测试",
    Body:    "内容",
})
```

### 方法 3：使用底层 Publish 方法

```go
// 直接指定 topic 和 payload
err := publisher.Publish(ctx, mq.TopicEmail, mq.EmailPayload{
    To:      "user@example.com",
    Subject: "测试",
    Body:    "内容",
})
```

### 错误处理

```go
err := publisher.PublishEmail(ctx, payload)
if err != nil {
    // 发布失败的常见原因：
    // 1. Redis/MySQL 连接断开
    // 2. Payload 序列化失败
    // 3. Context 超时或取消
    logger.Log.Errorw("publish email failed", "err", err)
    
    // 根据业务场景决定是否需要重试或降级
    return fmt.Errorf("send email notification failed: %w", err)
}
```

---

## 消费消息

### 自动消费

消息队列启动后会自动消费，无需手动干预：

```go
// cmd/server/main.go
mqRouter, err := mq.NewRouter(&cfg.MQ, rdb, sqlDB)
if err != nil {
    log.Fatal(err)
}

// 在 goroutine 中启动消费者
go func() {
    if err := mqRouter.Run(ctx); err != nil {
        log.Error("mq router stopped", err)
    }
}()
```

### Handler 实现规范

```go
func handleEmail(msg *message.Message, payload EmailPayload) error {
    // 1. 业务逻辑处理
    err := sendEmailViaSMTP(payload.To, payload.Subject, payload.Body)
    if err != nil {
        // 2. 返回错误会触发自动重试（最多 3 次）
        return fmt.Errorf("smtp send failed: %w", err)
    }
    
    // 3. 记录日志（建议包含 msg.UUID 用于链路追踪）
    logger.Log.Infow("[mq] email sent successfully",
        "msg_uuid", msg.UUID,
        "to", payload.To,
    )
    
    // 4. 返回 nil 表示消费成功，消息会被 ACK
    return nil
}
```

### 重试机制

消息处理失败时会自动重试：

```
第 1 次失败 → 等待 100ms → 重试
第 2 次失败 → 等待 200ms → 重试
第 3 次失败 → 等待 400ms → 重试
第 4 次失败 → 进入死信队列 (mq.poison)
```

### 死信队列

超过重试次数的消息会进入 `mq.poison` 队列，可以手动排查：

```bash
# Redis 模式
redis-cli XRANGE mq.poison - +

# MySQL 模式
SELECT * FROM watermill_messages WHERE topic = 'mq.poison';
```

---

## 高级特性

### 1. 并发消费

每个 topic 可以配置多个消费者并发处理：

```yaml
mq:
  topic_concurrency:
    email: 5  # 5 个消费者并发处理邮件
```

等价于：

```
email_handler_0 ──┐
email_handler_1 ──┤
email_handler_2 ──┼──> TopicEmail
email_handler_3 ──┤
email_handler_4 ──┘
```

### 2. 消息裁剪（Redis 模式）

防止 Redis Stream 无限增长：

```yaml
mq:
  default_max_len: 1000      # 全局默认
  topic_max_len:
    email: 100               # 邮件队列最多 100 条
    stats: 50000             # 统计队列最多 50000 条
  trim_interval: 600         # 每 10 分钟裁剪一次
```

裁剪策略：
- 使用 `XTRIM MAXLEN ~ <max_len>` 近似裁剪（性能更好）
- 只删除已消费的消息，不影响未消费消息

### 3. 链路追踪

消息 UUID 会复用 HTTP 请求的 `request_id`：

```go
// HTTP 请求
GET /api/users/123
X-Request-ID: abc-123-def

// 发布消息
publisher.PublishEmail(ctx, payload)
// → 消息 UUID = "abc-123-def"

// 消费消息
func handleEmail(msg *message.Message, payload EmailPayload) error {
    logger.Log.Infow("processing email", "msg_uuid", msg.UUID)
    // → msg_uuid = "abc-123-def"
}
```

可以通过 `request_id` 关联 HTTP 日志和 MQ 日志。

### 4. Context 传递

Publisher 会透传 Context 的超时和取消信号：

```go
// 设置 5 秒超时
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// 如果 5 秒内发布失败，会返回 context.DeadlineExceeded
err := publisher.PublishEmail(ctx, payload)
```

---

## 最佳实践

### 1. 消息幂等性

消息可能被重复消费，Handler 应该实现幂等：

```go
func handleEmail(msg *message.Message, payload EmailPayload) error {
    // 方案 1：使用消息 UUID 去重
    key := fmt.Sprintf("email:sent:%s", msg.UUID)
    exists, _ := redis.Exists(ctx, key).Result()
    if exists {
        return nil // 已处理过，跳过
    }
    
    // 发送邮件
    err := sendEmail(payload)
    if err != nil {
        return err
    }
    
    // 标记已处理（24 小时过期）
    redis.Set(ctx, key, "1", 24*time.Hour)
    return nil
}
```

### 2. 错误分类

区分可重试错误和不可重试错误：

```go
func handleSMS(msg *message.Message, payload SMSPayload) error {
    err := sendSMS(payload.Phone, payload.Content)
    if err != nil {
        // 不可重试错误：手机号格式错误
        if errors.Is(err, ErrInvalidPhone) {
            logger.Log.Warnw("invalid phone, skip retry", "phone", payload.Phone)
            return nil // 返回 nil 避免无意义重试
        }
        
        // 可重试错误：网络超时
        if errors.Is(err, ErrTimeout) {
            return err // 返回错误触发重试
        }
        
        return err
    }
    return nil
}
```

### 3. 监控指标

建议监控以下指标：

```go
// 发布成功率
metrics.Counter("mq.publish.success", "topic", topic)
metrics.Counter("mq.publish.failure", "topic", topic)

// 消费延迟
start := time.Now()
err := handler(msg)
metrics.Histogram("mq.consume.duration", time.Since(start), "topic", topic)

// 死信队列长度
metrics.Gauge("mq.poison.length", getPoisonQueueLength())
```

### 4. 优雅停机

确保消息处理完成后再退出：

```go
// cmd/server/app.go
func (a *App) Shutdown(ctx context.Context) error {
    // 1. 停止接收新消息
    if err := a.mqRouter.Close(); err != nil {
        logger.Log.Warnw("mq router close failed", "err", err)
    }
    
    // 2. 等待正在处理的消息完成（最多等待 shutdownTimeout）
    <-ctx.Done()
    
    // 3. 关闭发布者
    if err := a.mqPublisher.Close(); err != nil {
        logger.Log.Warnw("mq publisher close failed", "err", err)
    }
    
    return nil
}
```

### 5. 配置建议

| 场景 | 并发度 | 队列长度 | 裁剪间隔 |
|------|--------|----------|----------|
| 邮件发送 | 3-5 | 100-500 | 600s |
| 短信发送 | 1-2 | 1000-5000 | 1800s |
| 数据统计 | 5-10 | 10000-50000 | 3600s |
| 实时通知 | 3-5 | 500-1000 | 600s |

---

## 故障排查

### 问题 1：消息发布失败

**现象**：`publisher.PublishEmail()` 返回错误

**排查步骤**：

1. 检查 Redis/MySQL 连接：
   ```bash
   # Redis
   redis-cli PING
   
   # MySQL
   mysql -h 127.0.0.1 -u root -p -e "SELECT 1"
   ```

2. 检查配置：
   ```yaml
   mq:
     driver: redis  # 确认 driver 正确
   ```

3. 查看日志：
   ```bash
   grep "mq publisher" logs/error.log
   ```

### 问题 2：消息未被消费

**现象**：消息发布成功，但 Handler 未执行

**排查步骤**：

1. 确认 Router 已启动：
   ```go
   go func() {
       if err := mqRouter.Run(ctx); err != nil {
           log.Error("mq router stopped", err)
       }
   }()
   ```

2. 检查消费者组（Redis 模式）：
   ```bash
   redis-cli XINFO GROUPS silk_route.email
   ```

3. 查看消息积压：
   ```bash
   # Redis
   redis-cli XLEN silk_route.email
   
   # MySQL
   SELECT COUNT(*) FROM watermill_messages WHERE topic = 'silk_route.email';
   ```

### 问题 3：消息重复消费

**现象**：同一条消息被处理多次

**原因**：
- Handler 返回错误触发重试
- 消费者崩溃后重新消费未 ACK 的消息

**解决方案**：
- 实现幂等性（见最佳实践）
- 检查 Handler 是否正确返回 `nil`

### 问题 4：死信队列堆积

**现象**：`mq.poison` 队列消息数量持续增长

**排查步骤**：

1. 查看死信消息：
   ```bash
   redis-cli XRANGE mq.poison - + COUNT 10
   ```

2. 分析失败原因：
   ```go
   // 在 Handler 中记录详细错误
   logger.Log.Errorw("handler failed",
       "msg_uuid", msg.UUID,
       "payload", payload,
       "err", err,
   )
   ```

3. 修复 Handler 逻辑后，手动重放死信消息

### 问题 5：Redis 内存溢出

**现象**：Redis 内存使用率持续上升

**解决方案**：

1. 启用消息裁剪：
   ```yaml
   mq:
     default_max_len: 1000
     trim_interval: 600
   ```

2. 调整队列长度：
   ```yaml
   mq:
     topic_max_len:
       stats: 10000  # 减小队列长度
   ```

3. 增加消费者并发度：
   ```yaml
   mq:
     topic_concurrency:
       stats: 10  # 加快消费速度
   ```

---

## 附录

### A. 完整示例：用户注册流程

```go
// internal/service/user_service.go
type UserService struct {
    repo      repository.UserRepository
    publisher *mq.Publisher
}

func (s *UserService) Register(ctx context.Context, req RegisterRequest) error {
    // 1. 创建用户
    user := &model.User{
        Username: req.Username,
        Email:    req.Email,
        Phone:    req.Phone,
    }
    if err := s.repo.Create(ctx, user); err != nil {
        return err
    }
    
    // 2. 发送欢迎邮件（异步）
    go func() {
        ctx := context.Background()
        err := s.publisher.PublishEmail(ctx, mq.EmailPayload{
            To:      user.Email,
            Subject: "欢迎注册 Gonio",
            Body:    fmt.Sprintf("Hi %s, 欢迎加入！", user.Username),
        })
        if err != nil {
            logger.Log.Warnw("publish welcome email failed", "err", err)
        }
    }()
    
    // 3. 发送验证短信（异步）
    go func() {
        ctx := context.Background()
        code := generateVerifyCode()
        err := s.publisher.PublishSMS(ctx, mq.SMSPayload{
            Phone:   user.Phone,
            Content: fmt.Sprintf("验证码：%s", code),
        })
        if err != nil {
            logger.Log.Warnw("publish verify sms failed", "err", err)
        }
    }()
    
    // 4. 记录统计事件（异步）
    go func() {
        ctx := context.Background()
        err := s.publisher.PublishStats(ctx, mq.StatsPayload{
            Event:  "user.register",
            UserID: user.ID,
            Properties: map[string]any{
                "source": req.Source,
                "ip":     req.IP,
            },
        })
        if err != nil {
            logger.Log.Warnw("publish stats failed", "err", err)
        }
    }()
    
    return nil
}
```

### B. 消息类型注册表

当前已注册的消息类型：

| 短名称 | Topic | Payload | Handler |
|--------|-------|---------|---------|
| `email` | `silk_route.email` | `EmailPayload` | `handleEmail` |
| `sms` | `silk_route.sms` | `SMSPayload` | `handleSMS` |
| `stats` | `silk_route.stats` | `StatsPayload` | `handleStats` |

### C. 相关文件

```
internal/mq/
├── registry.go      # 消息类型注册表（核心）
├── types.go         # 消息类型定义（新增消息在此修改）
├── publisher.go     # 发布者
├── router.go        # 消费者路由
├── trimmer.go       # Redis 消息裁剪
└── logger.go        # Watermill 日志适配器
```

---

## 总结

通过统一注册表模式，Gonio 消息队列实现了：

✅ **开发效率**：新增消息类型只需修改 1 个文件  
✅ **类型安全**：编译时检查 Payload 类型  
✅ **高可用**：自动重试 + 死信队列  
✅ **高性能**：并发消费 + 消息裁剪  
✅ **可观测**：链路追踪 + 结构化日志  

如有问题，请查阅 [故障排查](#故障排查) 章节或联系开发团队。
