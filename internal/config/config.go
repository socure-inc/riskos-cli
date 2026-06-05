// Package config manages RiskOS credentials.
// Env vars take precedence over the stored file (CLIG: env vars > config file).
//
// Two separate keys:
//   - APIKey   → CLI/MCP Bearer token  (used for all /api/mcp/* endpoints)
//   - EvalKey  → Evaluation API Bearer token (used for /api/evaluation)
package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Credentials holds the RiskOS authentication material.
type Credentials struct {
	APIKey    string `json:"api_key"`              // CLI/MCP key — Bearer token for /api/mcp/*
	EvalKey   string `json:"eval_api_key,omitempty"` // Evaluation API key for /api/evaluation
	Namespace string `json:"namespace,omitempty"`
	Prod      bool   `json:"prod,omitempty"`
}

// Store manages credential persistence at a configurable directory path.
type Store struct {
	dir string
}

// NewStore returns a Store rooted at the given directory.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// DefaultStore returns a Store using the default ~/.riskos directory.
func DefaultStore() *Store {
	home, _ := os.UserHomeDir()
	return &Store{dir: filepath.Join(home, ".riskos")}
}

// CredFile returns the absolute path to the credentials file.
func (s *Store) CredFile() string {
	return filepath.Join(s.dir, "credentials.json")
}

// Load reads credentials from disk and overlays env vars.
// Returns nil (no error) if no API key is available.
func (s *Store) Load() (*Credentials, error) {
	c := &Credentials{}
	if data, err := os.ReadFile(s.CredFile()); err == nil {
		if err := json.Unmarshal(data, c); err != nil {
			log.Printf("warning: credentials file is malformed, ignoring: %v", err)
			c = &Credentials{}
		}
	}
	if v := os.Getenv("RISKOS_API_KEY"); v != "" {
		c.APIKey = v
	}
	if v := os.Getenv("RISKOS_EVAL_KEY"); v != "" {
		c.EvalKey = v
	}
	if c.APIKey == "" {
		return nil, nil
	}
	return c, nil
}

// Save writes credentials to disk with 0600 permissions.
func (s *Store) Save(c *Credentials) error {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	// Ensure the directory has correct permissions even if it already existed.
	if err := os.Chmod(s.dir, 0700); err != nil {
		return fmt.Errorf("chmod config dir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	return os.WriteFile(s.CredFile(), data, 0600)
}

// ─── package-level helpers (backward compat for non-test callers) ─────────────

// Load reads credentials using the default store.
func Load() (*Credentials, error) {
	return DefaultStore().Load()
}

// Save writes credentials using the default store.
func Save(c *Credentials) error {
	return DefaultStore().Save(c)
}
