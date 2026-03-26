package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"goflow/internal/config"
	"goflow/internal/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func InitMySQL(cfg *config.MySQLConfig, logCfg *config.LogConfig) (*gorm.DB, error) {
	// 设置超时默认值
	dialTimeout := cfg.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = 5
	}
	readTimeout := cfg.ReadTimeout
	if readTimeout <= 0 {
		readTimeout = 10
	}
	writeTimeout := cfg.WriteTimeout
	if writeTimeout <= 0 {
		writeTimeout = 10
	}
	pingTimeout := cfg.PingTimeout
	if pingTimeout <= 0 {
		pingTimeout = 3
	}

	// DSN 中加入连接与 I/O 超时参数
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC&timeout=%ds&readTimeout=%ds&writeTimeout=%ds",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database,
		dialTimeout, readTimeout, writeTimeout)

	// 根据配置文件设置 GORM 日志级别
	logLevel := parseGormLogLevel(logCfg.SQLLevel)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(logLevel),
		PrepareStmt:            cfg.PrepareStmt,            // 缓存预编译语句，减少重复 SQL 解析开销
		SkipDefaultTransaction: cfg.SkipDefaultTransaction, // 单条操作跳过默认事务包裹，提升约 30% 性能
	})
	if err != nil {
		return nil, fmt.Errorf("mysql connect failed: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB failed: %w", err)
	}

	// 连接池配置
	maxIdleConns := cfg.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 10
	}
	maxOpenConns := cfg.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = 100
	}
	maxLifetime := cfg.MaxLifetime
	if maxLifetime <= 0 {
		maxLifetime = 3600
	}
	connMaxIdleTime := cfg.ConnMaxIdleTime
	if connMaxIdleTime <= 0 {
		connMaxIdleTime = 600
	}

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(maxLifetime) * time.Second)
	sqlDB.SetConnMaxIdleTime(time.Duration(connMaxIdleTime) * time.Second)

	// 带超时的健康检查
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(pingTimeout)*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("mysql ping failed: %w", err)
	}

	logger.Log.Info("mysql connected successfully",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Database),
		zap.Int("max_idle_conns", maxIdleConns),
		zap.Int("max_open_conns", maxOpenConns),
		zap.Int("max_lifetime_s", maxLifetime),
		zap.Int("conn_max_idle_time_s", connMaxIdleTime),
		zap.Bool("prepare_stmt", cfg.PrepareStmt),
		zap.Bool("skip_default_transaction", cfg.SkipDefaultTransaction),
	)
	return db, nil
}

// parseGormLogLevel 将配置字符串转为 gormlogger.LogLevel
func parseGormLogLevel(level string) gormlogger.LogLevel {
	switch strings.ToLower(level) {
	case "silent":
		return gormlogger.Silent
	case "error":
		return gormlogger.Error
	case "warn", "warning":
		return gormlogger.Warn
	case "info":
		return gormlogger.Info
	default:
		return gormlogger.Warn
	}
}

// CloseMySQL 关闭 MySQL 连接，接收显式参数避免全局状态
func CloseMySQL(db *gorm.DB) {
	if db != nil {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}
}
