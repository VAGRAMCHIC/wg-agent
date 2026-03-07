package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/VAGRAMCHIC/wg-agent/internal/api"
	"github.com/VAGRAMCHIC/wg-agent/internal/wireguard"
	"github.com/VAGRAMCHIC/wg-agent/pkg/logger"
)

func main() {
	iface := os.Getenv("WG_INTERFACE")
	if iface == "" {
		iface = "wg0"
	}
	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = "log"
	}

	wgManager, err := wireguard.New(iface)
	if err != nil {
		log.Fatal(err)
	}

	log, err := logger.New(logPath)
	handler := api.New(wgManager)

	r := chi.NewRouter()
	r.Use(api.AuthMiddleware)

	r.Post("/peer", handler.AddPeer)
	r.Delete("/peer", handler.DeletePeer)
	r.Get("/peers", handler.ListPeers)
	r.Post("/peer/auto", handler.AddPeerAuto)
	r.Post("/peer/auto/qr", handler.QRPeerAuto)
	r.Get("/health", handler.Health)

	log.Info(nil, "wg-agent started on :8050", map[string]interface{}{
		"curr_date": time.Now(),
	})

	if err := http.ListenAndServe(":8050", r); err != nil {
		log.Fatal(nil, err.Error(), map[string]interface{}{
			"curr_date": time.Now(),
		})
	}
}
