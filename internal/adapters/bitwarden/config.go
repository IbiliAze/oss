package bitwarden

import (
	"errors"
	"time"
)

type Config struct {
	APIURL       string        `koanf:"api_url"`
	OrgID        string        `koanf:"org_id"`
	IdentityURL  string        `koanf:"identity_url"`
	ProjectID    string        `koanf:"project_id"`
	AccessToken  string        `koanf:"access_token"`
	PollInterval time.Duration `koanf:"poll_interval"`
}

func (c Config) Validate() error {
	if c.AccessToken == "" {
		return errors.New("bitwarden: access token required")
	}
	if c.ProjectID == "" {
		return errors.New("bitwarden: project_id required")
	}
	return nil
}
