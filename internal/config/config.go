package config

import "github.com/IbiliAze/vaultlet/internal/adapters/bitwarden"

type Config struct {
	Listen    string           `koanf:"listen"`
	Backend   string           `koanf:"backend"`
	Bitwarden bitwarden.Config `koanf:"bitwarden"`
}

func Load() (Config, error) {
	return Config{}, nil
}
