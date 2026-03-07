package config

import "os"

type Config struct {
	Interface string
	Address   string
	Listen    string
	LogFile   string
}

func Load() Config {
	return Config{
		Interface: getEnv("WG_INTERFACE", "wg0"),
		Address:   getEnv("WG_ADDRESS", "10.10.0.1/24"),
		Listen:    getEnv("HTTP_LISTEN", ":8080"),
		LogFile:   getEnv("LOG_FILE", "/var/log/wg-agent/agent.log"),
	}
}

func getEnv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
