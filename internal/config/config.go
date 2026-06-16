// Package config manages RiskOS credentials.
//
// Resolution order (lowest to highest priority):
//  3. OS keyring via go-keyring — secure persistent store (base)
//  2. Environment variables (RISKOS_API_KEY, RISKOS_EVAL_KEY, RISKOS_BASE_URL) — override keyring
//  1. CLI flags (--base-url) — override everything (applied in cmd layer)
//
// Two separate keys:
//   - APIKey   → CLI/MCP Bearer token  (used for all /api/mcp/* endpoints)
//   - EvalKey  → Evaluation API Bearer token (used for /api/evaluation)
package config

import (
	"encoding/json"
	"log"
	"os"

	"github.com/zalando/go-keyring"
	"riskos/internal/debug"
)

const (
	ServiceName = "riskos-cli"
	KeyringUser = "credentials"
)

// Credentials holds the RiskOS authentication material.
type Credentials struct {
	APIKey    string `json:"api_key"`                // CLI/MCP key — Bearer token for /api/mcp/*
	EvalKey   string `json:"eval_api_key,omitempty"` // Evaluation API key for /api/evaluation
	Namespace string `json:"namespace,omitempty"`
	BaseURL   string `json:"base_url,omitempty"` // API base URL (empty = default sandbox)
}

// CredSource describes where credentials were loaded from.
type CredSource string

const (
	SourceKeyring CredSource = "keyring"
	SourceEnvVar  CredSource = "env"
	SourceNone    CredSource = "none"
)

// Store manages credential persistence via the OS keyring.
type Store struct{}

// NewStore returns a Store. The argument is ignored (kept for test compatibility).
func NewStore(_ string) *Store {
	return &Store{}
}

// DefaultStore returns a Store.
func DefaultStore() *Store {
	return &Store{}
}

// Load reads credentials with priority: env vars > keyring.
// Returns nil (no error) if no API key is available.
func (s *Store) Load() (*Credentials, error) {
	c := &Credentials{}

	debug.Log("reading credentials from keyring (service=%s)", ServiceName)
	if data, err := keyring.Get(ServiceName, KeyringUser); err == nil {
		debug.Log("keyring: found")
		if err := json.Unmarshal([]byte(data), c); err != nil {
			log.Printf("warning: keyring data is malformed, ignoring: %v", err)
			c = &Credentials{}
		}
	} else {
		debug.Log("keyring: not found (%v)", err)
	}

	// Env vars override keyring
	if v := os.Getenv("RISKOS_API_KEY"); v != "" {
		c.APIKey = v
		debug.Log("env override: RISKOS_API_KEY=***set***")
	}
	if v := os.Getenv("RISKOS_EVAL_KEY"); v != "" {
		c.EvalKey = v
		debug.Log("env override: RISKOS_EVAL_KEY=***set***")
	}
	if v := os.Getenv("RISKOS_BASE_URL"); v != "" {
		c.BaseURL = v
		debug.Log("env override: RISKOS_BASE_URL=%s", v)
	}

	if c.APIKey == "" {
		debug.Log("no API key found — returning nil credentials")
		return nil, nil
	}
	return c, nil
}

// Source returns where the active credentials originate from.
func (s *Store) Source() CredSource {
	if os.Getenv("RISKOS_API_KEY") != "" {
		return SourceEnvVar
	}
	if _, err := keyring.Get(ServiceName, KeyringUser); err == nil {
		return SourceKeyring
	}
	return SourceNone
}

// Save writes credentials to the OS keyring. If keyring is unavailable,
// logs a warning and returns nil (user must rely on env vars).
func (s *Store) Save(c *Credentials) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	debug.Log("writing credentials to keyring (service=%s)", ServiceName)
	if err := keyring.Set(ServiceName, KeyringUser, string(data)); err != nil {
		debug.Log("keyring write failed: %v", err)
		log.Printf("warning: keyring unavailable (%v), credentials not persisted — use RISKOS_API_KEY env var", err)
		return nil
	}
	debug.Log("keyring write: success")
	return nil
}

// Delete removes credentials from the keyring.
func (s *Store) Delete() error {
	debug.Log("deleting credentials from keyring (service=%s)", ServiceName)
	err := keyring.Delete(ServiceName, KeyringUser)
	if err != nil {
		debug.Log("keyring delete: %v", err)
	} else {
		debug.Log("keyring delete: success")
	}
	return nil
}
