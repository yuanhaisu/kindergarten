package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/app"
	"kindergarten-server/internal/config"
	pkglog "kindergarten-server/pkg/log"
)

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
	log.Printf("server starting on %s", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
