package middleware

import (
	"fmt"
	"strings"
	"time"
)

// ZitadelConfig centralizes the common authentication settings shared across services.
type ZitadelConfig struct {
	Issuer     string
	Audience   string
	ClientID   string
	ProjectID  string
	ClockSkew  time.Duration
	RequireTLS bool
}

// Validate checks the minimum fields required for a Zitadel-backed auth setup.
func (c ZitadelConfig) Validate() error {
	if strings.TrimSpace(c.Issuer) == "" {
		return fmt.Errorf("zitadel issuer is required")
	}

	if strings.TrimSpace(c.Audience) == "" {
		return fmt.Errorf("zitadel audience is required")
	}

	if strings.TrimSpace(c.ClientID) == "" {
		return fmt.Errorf("zitadel client ID is required")
	}

	if strings.TrimSpace(c.ProjectID) == "" {
		return fmt.Errorf("zitadel project ID is required")
	}

	if c.ClockSkew < 0 {
		return fmt.Errorf("zitadel clock skew cannot be negative")
	}

	return nil
}