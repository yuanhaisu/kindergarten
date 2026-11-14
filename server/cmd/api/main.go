package main

import (
	"log"
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
