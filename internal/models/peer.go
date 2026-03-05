package models

type PeerRequest struct {
	PublicKey string `json:"public_key"`
	AllowedIPs string `json:"allowed_ips,omitempty"` // optional для auto IP
}