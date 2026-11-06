package main

import (
	"context"
	"demo/app/internal/bootstrap"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	// log.Fatalf 会跳过 defer，所以资源释放必须放在 run 内部
	if err := run(); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}

func run() error {
	// 收到 SIGINT/SIGTERM 时关闭 ctx 进而过触发优雅退出，程序不再只能被 kill
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := bootstrap.Init(ctx, debugMode())
	if err != nil {
		return err
	}
	defer func() {
		if err := app.Close(); err != nil {
			log.Printf("释放资源失败: %v", err)
		}
	}()

	srv := &http.Server{
		Addr:    addr(),
		Handler: newRoutes(app),
		// ReadHeaderTimeout 防止恶意客户端只发一个请求头占住连接（slowloris）
		ReadHeaderTimeout: 5 * time.Second,
	}

	// ctx 关闭时开始优雅关停：等存量请求处理完，最多等 5s
	shutdownDone := make(chan error, 1)
	go func() {
		<-ctx.Done()
		log.Printf("收到退出信号，开始优雅关停…")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		shutdownDone <- srv.Shutdown(shutdownCtx)
	}()

	log.Printf("HTTP 服务已启动: %s", srv.Addr)
	// Shutdown 成功后 ListenAndServe 返回 ErrServerClosed，这是正常路径不是错误
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	// 等优雅关停结束，保证 defer app.Close() 在所有连接释放之后执行
	if err := <-shutdownDone; err != nil {
		log.Printf("优雅关停失败: %v", err)
	}
	return nil
}

// newRoutes 装配路由。app 携带 DB/Redis，后续 handler 可从这里取依赖。
func newRoutes(app *bootstrap.App) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthz)
	return mux
}

// healthz 存活探针：只校验进程本身是否启动成功、能否正常响应 HTTP，
// 不探测 MySQL/Redis 等依赖——依赖短暂抖动时不至于被 LB/K8s 误判重启或摘除。
// 若以后需要"依赖也健康才算可用"的读探针，单独加 /readyz 再做依赖检查。
func healthz(w http.ResponseWriter, r *http.Request) {
	// 明确不缓存，LB/K8s 每次探测都要拿到实时结果
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// addr 从 DEMO_ADDR 读取监听地址，缺省 :8080。
func addr() string {
	if v := strings.TrimSpace(os.Getenv("DEMO_ADDR")); v != "" {
		return v
	}
	return ":8080"
}

// debugMode 从 DEMO_DEBUG 读取，开启后 GORM 打印 SQL 日志。
// 只认 true/false，其他值启动即失败，避免手滑写成 "ture" 时静默按 false 跑。
func debugMode() (debug bool) {
	v := strings.TrimSpace(os.Getenv("DEMO_DEBUG"))
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		log.Fatalf("环境变量 DEMO_DEBUG=%q 不是合法布尔值（true/false）", v)
	}
	return b
}
