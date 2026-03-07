package api

import (
	"encoding/json"
	"net/http"

	"github.com/VAGRAMCHIC/wg-agent/internal/models"
	"github.com/VAGRAMCHIC/wg-agent/internal/wireguard"
	"github.com/VAGRAMCHIC/wg-agent/pkg/logger"
	qrcode "github.com/skip2/go-qrcode"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Handler struct {
	wg  *wireguard.Manager
	log *logger.Logger
}

func New(wg *wireguard.Manager, log *logger.Logger) *Handler {
	return &Handler{wg: wg, log: log}
}

func generateKeys() (privateKey string, publicKey string, err error) {
	priv, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", "", err
	}
	return priv.String(), priv.PublicKey().String(), nil
}

func generateWGConf(privKey, addr, serverPubKey, endpoint string) string {
	return `[Interface]
PrivateKey = ` + privKey + `
Address = ` + addr + `
DNS = 1.1.1.1

[Peer]
PublicKey = ` + serverPubKey + `
Endpoint = ` + endpoint + `
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
`
}

func (h *Handler) QRPeerAuto(w http.ResponseWriter, r *http.Request) {
	var req models.PeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	privKey, pubKey, err := generateKeys()
	if err != nil {
		h.log.Error(nil, "failed_to_generate-generateKeys", map[string]interface{}{
			"error": err.Error(),
			"func":  "handler.QRPeerAuto",
		})
		http.Error(w, err.Error(), 500)
		return
	}

	ip, err := h.wg.AddPeerAuto(pubKey)
	if err != nil {
		h.log.Error(nil, "failed_to_generate-AddPeerAuto", map[string]interface{}{
			"error": err.Error(),
			"func":  "handler.QRPeerAuto",
		})
		http.Error(w, err.Error(), 500)
		return
	}

	conf := generateWGConf(privKey, ip.String()+"/32", h.wg.ServerPublicKey(), h.wg.Endpoint())

	png, err := qrcode.Encode(conf, qrcode.Medium, 256)
	if err != nil {
		h.log.Error(nil, "failed_to_generateWGConf", map[string]interface{}{
			"error": err.Error(),
			"func":  "handler.QRPeerAuto",
		})
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

func (h *Handler) AddPeer(w http.ResponseWriter, r *http.Request) {
	var req models.PeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error(nil, "failed_AddPeer_NewDecoder", map[string]interface{}{
			"error": err.Error(),
			"func":  "handler.AddPeer",
		})
		http.Error(w, err.Error(), 400)
		return
	}
	if req.AllowedIPs == "" {
		http.Error(w, "allowed_ips required", 400)
		return
	}

	if err := h.wg.AddPeer(req.PublicKey, req.AllowedIPs); err != nil {
		h.log.Error(nil, "failed_AddPeer_AddPeer", map[string]interface{}{
			"error": err.Error(),
			"func":  "handler.AddPeer",
		})
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) DeletePeer(w http.ResponseWriter, r *http.Request) {
	var req models.PeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error(nil, "failed_to_DeletePeer", map[string]interface{}{
			"error": err.Error(),
			"func":  "handler.DeletePeer",
		})
		http.Error(w, err.Error(), 500)
		return
	}

	if err := h.wg.RemovePeer(req.PublicKey); err != nil {
		h.log.Error(nil, "failed_to_DeletePeer_RemovePeer", map[string]interface{}{
			"error": err.Error(),
			"func":  "handler.RemovePeer",
		})
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListPeers(w http.ResponseWriter, r *http.Request) {
	peers, err := h.wg.ListPeers()
	if err != nil {
		h.log.Error(nil, "failed_to_ListPeers", map[string]interface{}{
			"error": err.Error(),
			"func":  "handler.ListPeers",
		})
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(peers)
}

// AddPeerAuto allocates IP automatically
func (h *Handler) AddPeerAuto(w http.ResponseWriter, r *http.Request) {
	var req models.PeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error(nil, "failed_to_AddPeerAuto_Decode", map[string]interface{}{
			"error": err.Error(),
			"func":  "handler.AddPeerAuto",
		})
		http.Error(w, err.Error(), 400)
		return
	}

	ip, err := h.wg.AddPeerAuto(req.PublicKey)
	if err != nil {
		h.log.Error(nil, "failed_to_AddPeerAuto", map[string]interface{}{
			"error": err.Error(),
			"func":  "handler.AddPeerAuto",
		})
		http.Error(w, err.Error(), 500)
		return
	}

	resp := map[string]string{"assigned_ip": ip.String()}
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
