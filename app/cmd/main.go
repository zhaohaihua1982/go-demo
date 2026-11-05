package main

import (
	"context"
	"demo/app/internal/bootstrap"
	"log"
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
	return nil
}
