package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"demo/conf"
)

// InitCache 建立 redis 客户端并做启动探活。
// 调用方负责在退出时调用 Close 释放连接。
func InitCache(ctx context.Context, cfg *conf.Redis) (*redis.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("redis 配置为空")
	}

	addr := cfg.RedisHost + ":" + cfg.RedisPort
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Username:     cfg.RedisUsername,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDbName,
		PoolSize:     20,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("连接 redis 失败(%s): %w", addr, err)
	}

	return client, nil
}

// Close 释放 redis 连接。
func Close(client *redis.Client) error {
	if client == nil {
		return nil
	}
	return client.Close()
}
