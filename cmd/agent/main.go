package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/VAGRAMCHIC/wg-agent/internal/api"
	"github.com/VAGRAMCHIC/wg-agent/internal/wireguard"
)

func main() {
	iface := os.Getenv("WG_INTERFACE")
	if iface == "" {
		iface = "wg0"
	}

	wgManager, err := wireguard.New(iface)
	if err != nil {
		log.Fatal(err)
	}

	handler := api.New(wgManager)

	r := chi.NewRouter()
	r.Use(api.AuthMiddleware)

	r.Post("/peer", handler.AddPeer)
	r.Delete("/peer", handler.DeletePeer)
	r.Get("/peers", handler.ListPeers)
	r.Post("/peer/auto", handler.AddPeerAuto)
	r.Post("/peer/auto/qr", handler.QRPeerAuto)
	r.Get("/health", handler.Health)

	log.Println("wg-agent started on :8080")

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}