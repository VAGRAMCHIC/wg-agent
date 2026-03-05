package api

import (
	"encoding/json"
	"net/http"

	qrcode "github.com/skip2/go-qrcode"
	"github.com/VAGRAMCHIC/wg-agent/internal/models"
	"github.com/VAGRAMCHIC/wg-agent/internal/wireguard"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Handler struct {
	wg *wireguard.Manager
}

func New(wg *wireguard.Manager) *Handler {
	return &Handler{wg: wg}
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
		http.Error(w, err.Error(), 500)
		return
	}

	ip, err := h.wg.AddPeerAuto(pubKey)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	conf := generateWGConf(privKey, ip.String()+"/32", h.wg.ServerPublicKey(), h.wg.Endpoint())

	png, err := qrcode.Encode(conf, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

func (h *Handler) AddPeer(w http.ResponseWriter, r *http.Request) {
	var req models.PeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if req.AllowedIPs == "" {
		http.Error(w, "allowed_ips required", 400)
		return
	}

	if err := h.wg.AddPeer(req.PublicKey, req.AllowedIPs); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) DeletePeer(w http.ResponseWriter, r *http.Request) {
	var req models.PeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if err := h.wg.RemovePeer(req.PublicKey); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListPeers(w http.ResponseWriter, r *http.Request) {
	peers, err := h.wg.ListPeers()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(peers)
}

// AddPeerAuto allocates IP automatically
func (h *Handler) AddPeerAuto(w http.ResponseWriter, r *http.Request) {
	var req models.PeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	ip, err := h.wg.AddPeerAuto(req.PublicKey)
	if err != nil {
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

