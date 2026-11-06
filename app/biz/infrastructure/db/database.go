package db

import (
	"context"
	"fmt"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"demo/conf"
)

// InitMySQL 建立 MySQL 连接池并做启动探活。
// debug 为 true 时打印 SQL；调用方负责在退出时调用 Close 释放连接。
//
// 注意：返回的是 GORM 根连接，任何派生调用（WithContext/Session）都会
// 创建新会话实例。建议所有使用处先 WithContext 取会话（本包目的之一是把
// 调用方 context 传给驱动，实现超时/取消传播）；直接在根连接上链式调用
// 拿不到 ctx 传播。
func InitMySQL(cfg *conf.MySql, debug bool) (*gorm.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("mysql 配置为空")
	}

	// 用驱动自带的 Config 拼 DSN，密码里的 @ / : 等字符会被正确转义
	dsn := mysqldriver.NewConfig()
	dsn.User = cfg.UserName
	dsn.Passwd = cfg.Password
	dsn.Net = "tcp"
	dsn.Addr = cfg.DbHost + ":" + cfg.DbPort
	dsn.DBName = cfg.DbName
	dsn.ParseTime = true
	dsn.Timeout = 3 * time.Second
	dsn.ReadTimeout = 10 * time.Second
	dsn.WriteTimeout = 10 * time.Second
	dsn.Params = map[string]string{"charset": cfg.Charset}

	logLevel := logger.Warn
	if debug {
		logLevel = logger.Info
	}

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		DSNConfig:                 dsn,
		DefaultStringSize:         256,
		DisableDatetimePrecision:  true, // MySQL 5.6 之前的数据库不支持 datetime 精度
		DontSupportRenameIndex:    true, // 重命名索引时采用删除并新建的方式
		DontSupportRenameColumn:   true, // 用 `change` 重命名列
		SkipInitializeWithVersion: false,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("连接 mysql 失败: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("获取 mysql 连接池失败: %w", err)
	}

	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // 需小于 MySQL wait_timeout
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("mysql 探活失败: %w", err)
	}

	return gormDB.Set("gorm:table_options", "charset=utf8mb4"), nil
}

// Close 释放 MySQL 连接池。
func Close(gormDB *gorm.DB) error {
	if gormDB == nil {
		return nil
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("获取 mysql 连接池失败: %w", err)
	}
	return sqlDB.Close()
}
