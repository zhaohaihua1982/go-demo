package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"log"

	"demo/pkg/i18n"

	"demo/app/biz/infrastructure/cache"
	"demo/app/biz/infrastructure/db"
	"demo/conf"
)

// App 持有进程内所有需要释放的资源。
type App struct {
	Conf  *conf.Conf
	DB    *gorm.DB
	Redis *redis.Client
}

// Init 按依赖顺序初始化：配置 → MySQL → Redis。
// 任一步失败都会先释放已建立的资源，再由调用方决定退出方式。
func Init(ctx context.Context, debug bool) (*App, error) {
	if err := conf.InitConfig(); err != nil {
		return nil, err
	}

	app := &App{Conf: conf.Config}

	// 默认语言：yaml 显式配置优先于 DEMO_LANG 环境变量
	if app.Conf.I18n != nil && app.Conf.I18n.DefaultLang != "" {
		if lang, ok := i18n.ParseLang(app.Conf.I18n.DefaultLang); ok {
			i18n.SetDefaultLang(lang)
		} else {
			log.Printf("配置 i18n.defaultLang=%q 不是已注册语言，回退默认链", app.Conf.I18n.DefaultLang)
		}
	}

	gormDB, err := db.InitMySQL(app.Conf.MySql["default"], debug)
	if err != nil {
		return nil, err
	}
	app.DB = gormDB

	redisClient, err := cache.InitCache(ctx, app.Conf.Redis)
	if err != nil {
		_ = app.Close() // 回滚已建立的 MySQL 连接
		return nil, err
	}
	app.Redis = redisClient

	return app, nil
}

// Close 释放所有资源，多个错误会被合并返回。
func (a *App) Close() error {
	if a == nil {
		return nil
	}

	var errs []error
	if err := cache.Close(a.Redis); err != nil {
		errs = append(errs, fmt.Errorf("关闭 redis: %w", err))
	}
	if err := db.Close(a.DB); err != nil {
		errs = append(errs, fmt.Errorf("关闭 mysql: %w", err))
	}
	return errors.Join(errs...)
}
