package main

import (
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/app"
	"kindergarten-server/internal/config"
	pkglog "kindergarten-server/pkg/log"
)

const shutdownTimeout = 10 * time.Second

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/app.ini"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	pkglog.Init(cfg.Log.Level, cfg.Log.Format == "json")

	// release 模式下 CORS 通配符等于向任意站点放行跨域读取；只告警不阻断，
	// 避免未知部署因配置检查而无法启动。
	if warning := cfg.App.CORSConfigWarning(); warning != "" {
		slog.Warn(warning, "gin_mode", cfg.App.GinMode, "cors_origins", cfg.App.CORSOrigins)
	}

	if cfg.App.GinMode != "" {
		gin.SetMode(cfg.App.GinMode)
	}

	engine, err := app.InjectEngine(cfg)
	if err != nil {
		log.Fatalf("failed to inject engine: %v", err)
	}

	addr := cfg.App.HTTPAddr
	if addr == "" {
		addr = ":8080"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", addr, err)
	}

	srv := &http.Server{
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	log.Printf("server starting on %s", addr)
	if err := app.ServeWithGracefulShutdown(srv, ln, shutdownTimeout, signals); err != nil {
		log.Fatalf("server error: %v", err)
	}
	log.Printf("server stopped")
}
