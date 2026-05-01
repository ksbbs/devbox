package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"devbox/internal/config"
	"devbox/internal/mirror"
	"devbox/internal/server"

	_ "devbox/internal/mirror"
)

func main() {
	configPath := flag.String("c", "configs/devbox.yaml", "config file path")
	frontDir := flag.String("f", "web/dist", "frontend dist directory")
	flag.Parse()

	if v := os.Getenv("DEVBOX_CONFIG"); v != "" {
		*configPath = v
	}
	if v := os.Getenv("DEVBOX_FRONTEND_DIR"); v != "" {
		*frontDir = v
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	log.Printf("Loaded config: %d mirrors, git proxy=%v, port=%d",
		len(cfg.Mirrors), cfg.GitProxy.Enabled, cfg.Server.Port)

	for _, m := range mirror.All() {
		mCfg := cfg.Mirrors[m.Name()]
		status := "disabled"
		if mCfg.Enabled {
			status = "enabled"
		}
		log.Printf("  Mirror %s: %s → %s", m.Name(), status, mCfg.Upstream)
	}

	srv, err := server.New(cfg, *frontDir)
	if err != nil {
		log.Fatalf("init server: %v", err)
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("Shutting down...")
		srv.Close()
		os.Exit(0)
	}()

	if err := srv.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}