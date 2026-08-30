package config

import (
	"strings"

	"github.com/IbiliAze/vaultlet/internal/adapters/bitwarden"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/dotenv"
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

	// 1. Base config
	if err := k.Load(file.Provider("vaultlet.yaml"), yaml.Parser()); err != nil {
		return Config{}, err
	}

	// 2. .env overrides YAML
	if err := k.Load(file.Provider(".env"), dotenv.ParserEnv(
		"VAULTLET_",
		".",
		func(s string) string {
			s = strings.ToLower(strings.TrimPrefix(s, "VAULTLET_"))
			return strings.ReplaceAll(s, "__", ".")
		},
	)); err != nil {
		return Config{}, err
	}

	// 3. Real environment variables override everything
	if err := k.Load(env.Provider("VAULTLET_", ".", func(s string) string {
		s = strings.ToLower(strings.TrimPrefix(s, "VAULTLET_"))
		return strings.ReplaceAll(s, "__", ".")
	}), nil); err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
