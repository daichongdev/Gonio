package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	MySQL  MySQLConfig  `mapstructure:"mysql"`
	Redis  RedisConfig  `mapstructure:"redis"`
	JWT    JWTConfig    `mapstructure:"jwt"`
	Log    LogConfig    `mapstructure:"log"`
	MQ     MQConfig     `mapstructure:"mq"`
}

// MQConfig 消息队列配置
type MQConfig struct {
	Driver           string         `mapstructure:"driver"`            // redis 或 mysql
	ConsumerGroup    string         `mapstructure:"consumer_group"`    // Redis Streams 消费者组名
	TopicConcurrency map[string]int `mapstructure:"topic_concurrency"` // 按 topic 配置并发消费者数，未配置默认为 1
	DefaultMaxLen    int64          `mapstructure:"default_max_len"`   // stream 默认最大长度，0 为不限制
	TopicMaxLen      map[string]int `mapstructure:"topic_max_len"`     // 按 topic 单独设置最大长度
	TrimInterval     int            `mapstructure:"trim_interval"`     // 定期 XTRIM 间隔（秒），0 为不启用
}

type ServerConfig struct {
	Port            int      `mapstructure:"port"`
	Mode            string   `mapstructure:"mode"`
	ReadTimeout     int      `mapstructure:"read_timeout"`
	WriteTimeout    int      `mapstructure:"write_timeout"`
	TrustedProxies  []string `mapstructure:"trusted_proxies"`
	AutoMigrate     bool     `mapstructure:"auto_migrate"`
	CacheExpire     int      `mapstructure:"cache_expire"`     // 商品详情缓存过期时间(秒)
	ShutdownTimeout int      `mapstructure:"shutdown_timeout"` // 优雅停机超时时间(秒)
	CORSOrigins     []string `mapstructure:"cors_origins"`     // 允许的跨域源，空则允许所有
}

type MySQLConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	Database        string `mapstructure:"database"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxLifetime     int    `mapstructure:"max_lifetime"`
	ConnMaxIdleTime int    `mapstructure:"conn_max_idle_time"` // 空闲连接最大存活时间(秒)
	DialTimeout     int    `mapstructure:"dial_timeout"`       // 连接超时(秒)
	ReadTimeout     int    `mapstructure:"read_timeout"`       // 读超时(秒)
	WriteTimeout    int    `mapstructure:"write_timeout"`      // 写超时(秒)
	PingTimeout     int    `mapstructure:"ping_timeout"`       // 探活超时(秒)

	// GORM 性能优化
	PrepareStmt            bool `mapstructure:"prepare_stmt"`             // 缓存预编译语句，减少 SQL 解析开销
	SkipDefaultTransaction bool `mapstructure:"skip_default_transaction"` // 非事务查询不包裹事务，提升约 30% 性能
}

type RedisConfig struct {
	Addr            string `mapstructure:"addr"`
	Password        string `mapstructure:"password"`
	DB              int    `mapstructure:"db"`
	PoolSize        int    `mapstructure:"pool_size"`
	MinIdleConns    int    `mapstructure:"min_idle_conns"`     // 最小空闲连接数
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`     // 最大空闲连接数
	PoolTimeout     int    `mapstructure:"pool_timeout"`       // 获取连接池超时(秒)
	DialTimeout     int    `mapstructure:"dial_timeout"`       // 连接超时(秒)
	ReadTimeout     int    `mapstructure:"read_timeout"`       // 读超时(秒)
	WriteTimeout    int    `mapstructure:"write_timeout"`      // 写超时(秒)
	ConnMaxIdleTime int    `mapstructure:"conn_max_idle_time"` // 空闲连接最大存活时间(秒)
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`  // 连接最大存活时间(秒)
	PingTimeout     int    `mapstructure:"ping_timeout"`       // 探活超时(秒)

	// 自动重试
	MaxRetries      int `mapstructure:"max_retries"`       // 命令失败最大重试次数(默认3)
	MinRetryBackoff int `mapstructure:"min_retry_backoff"` // 最小重试退避时间(毫秒, 默认8)
	MaxRetryBackoff int `mapstructure:"max_retry_backoff"` // 最大重试退避时间(毫秒, 默认512)
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire int    `mapstructure:"expire"`
}

type LogConfig struct {
	Mode       string `mapstructure:"mode"`
	Level      string `mapstructure:"level"`
	SQLLevel   string `mapstructure:"sql_level"`
	LogDir     string `mapstructure:"log_dir"`     // 日志目录，空则不写文件；warn.log/error.log 写入此目录
	MaxSize    int    `mapstructure:"max_size"`    // 单文件最大 MB，默认 100
	MaxBackups int    `mapstructure:"max_backups"` // 保留旧文件数，默认 7
	MaxAge     int    `mapstructure:"max_age"`     // 保留天数，默认 30
	Compress   bool   `mapstructure:"compress"`    // 是否压缩归档
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config failed: %w", err)
	}

	// 加载环境覆盖配置
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}
	v.SetConfigName("config." + env)
	if err := v.MergeInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("merge env config failed: %w", err)
		}
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	return &cfg, nil
}

// Validate 校验配置必填项和基本约束，启动时 fail-fast。
func (c *Config) Validate() error {
	var errs []string

	// Server
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		errs = append(errs, "server.port must be between 1 and 65535")
	}

	// MySQL
	if c.MySQL.Host == "" {
		errs = append(errs, "mysql.host is required")
	}
	if c.MySQL.Database == "" {
		errs = append(errs, "mysql.database is required")
	}

	// Redis
	if c.Redis.Addr == "" {
		errs = append(errs, "redis.addr is required")
	}

	// JWT
	if c.JWT.Secret == "" {
		errs = append(errs, "jwt.secret is required")
	}
	if c.JWT.Secret == "gonio-secret-key-change-me" {
		errs = append(errs, "jwt.secret must be changed from the default value")
	}

	// MQ
	switch c.MQ.Driver {
	case "redis", "mysql":
		// valid
	case "":
		errs = append(errs, "mq.driver is required (redis or mysql)")
	default:
		errs = append(errs, fmt.Sprintf("mq.driver '%s' is not supported (use redis or mysql)", c.MQ.Driver))
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}
