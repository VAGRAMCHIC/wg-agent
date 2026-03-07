package models

type Config struct {
	PublicKey  string `json:"public_key"`
	AllowedIPs string `json:"allowed_ips"`
	PrivateKey string `json:"private_key"`
}
