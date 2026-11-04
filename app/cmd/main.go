package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"demo/app/internal/bootstrap"
)

func main() {
	// log.Fatalf 会跳过 defer，所以资源释放必须放在 run 内部
	if err := run(); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}

func run() error {
	ctx := context.Background()

	app, err := bootstrap.Init(ctx, false)
	if err != nil {
		return err
	}
	defer func() {
		if err := app.Close(); err != nil {
			log.Printf("释放资源失败: %v", err)
		}
	}()

	sqlDB, err := app.DB.DB()
	if err != nil {
		return fmt.Errorf("获取 mysql 连接池失败: %w", err)
	}
	stats := sqlDB.Stats()

	// 冒烟检查：写一个带 TTL 的 key 再读回来
	const key = "demo:smoke"
	if err := app.Redis.Set(ctx, key, "dugu", time.Minute).Err(); err != nil {
		return fmt.Errorf("写入 redis 失败: %w", err)
	}
	value, err := app.Redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return fmt.Errorf("redis key %q 不存在", key)
	} else if err != nil {
		return fmt.Errorf("读取 redis 失败: %w", err)
	}

	log.Printf("启动成功: mysql=%s redis=%s:%s/%d",
		app.Conf.MySql["default"].DbName,
		app.Conf.Redis.RedisHost,
		app.Conf.Redis.RedisPort,
		app.Conf.Redis.RedisDbName,
	)
	log.Printf("mysql 连接池: open=%d idle=%d inUse=%d", stats.OpenConnections, stats.Idle, stats.InUse)
	log.Printf("redis 冒烟值: %s=%s (ttl 60s)", key, value)

	return nil
}
