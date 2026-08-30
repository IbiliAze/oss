package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	vaultletv1 "github.com/IbiliAze/vaultlet/api/gen/vaultlet/v1"
)

// metaJSON is the JSON projection of SecretMeta. It is declared here, not
// derived from the proto types, so that `-o json` is a stable contract for
// scripts even if the wire message grows fields.
type metaJSON struct {
	Key       string `json:"key"`
	Version   string `json:"version"`
	CreatedAt string `json:"created_at,omitempty"`
}

func toMetaJSON(m *vaultletv1.SecretMeta) metaJSON {
	out := metaJSON{Key: m.GetKey(), Version: m.GetVersion()}
	if ts := m.GetCreatedAt(); ts != nil {
		out.CreatedAt = ts.AsTime().UTC().Format(time.RFC3339)
	}
	return out
}

// writeJSON emits one value per line, so streaming commands and one-shot
// commands produce the same newline-delimited JSON.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// validateOutput rejects an unknown -o early, before any RPC is made.
func (o *options) validateOutput() error {
	switch o.output {
	case "text", "json":
		return nil
	default:
		return fmt.Errorf("unknown --output %q: want text or json", o.output)
	}
}
