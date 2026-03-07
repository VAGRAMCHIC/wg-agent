package main

import (
	"context"

	"github.com/VAGRAMCHIC/wg-agent/internal/config"
	"github.com/VAGRAMCHIC/wg-agent/internal/manager"
	"github.com/VAGRAMCHIC/wg-agent/internal/server"
	"github.com/VAGRAMCHIC/wg-agent/pkg/logger"
)

func main() {

	cfg := config.Load()

	log, err := logger.New(cfg.LogFile)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	log.Info(ctx, "wg-agent_starting", nil)

	mgr, err := manager.New(cfg.Interface, log)
	if err != nil {
		log.Fatal(ctx, "manager_init_failed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	err = mgr.EnsureInterface(ctx, cfg.Address)
	if err != nil {
		log.Fatal(ctx, "interface_init_failed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	srv := server.New(mgr, log)

	log.Info(ctx, "http_server_start", map[string]interface{}{
		"listen": cfg.Listen,
	})

	err = srv.Start(cfg.Listen)
	if err != nil {
		log.Fatal(ctx, "server_failed", map[string]interface{}{
			"error": err.Error(),
		})
	}
}
