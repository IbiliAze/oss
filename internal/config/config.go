package config

import (
	"strings"

	"github.com/IbiliAze/vaultlet/internal/adapters/bitwarden"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
)

type Config struct {
	Listen    string           `koanf:"listen"`
	Backend   string           `koanf:"backend"`
	Bitwarden bitwarden.Config `koanf:"bitwarden"`
}

func Load() (Config, error) {
	k := koanf.New(".")
	k.Load(file.Provider("vaultlet.yaml"), yaml.Parser())
	k.Load(env.Provider("VAULTLET_", ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(s, "VAULTLET_")), "_", ".")
	}), nil)

	var cfg Config
	k.Unmarshal("", &cfg)

	return cfg, nil
}
