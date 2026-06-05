package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSave_CreatesFileWith0600Permissions(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	creds := &Credentials{APIKey: "key1", Namespace: "acme"}

	if err := s.Save(creds); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(s.CredFile())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600, got %o", info.Mode().Perm())
	}
}

func TestLoad_ReturnsNilWhenNoFile(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "nonexistent"))

	// Clear any env vars that could interfere
	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	creds, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if creds != nil {
		t.Errorf("expected nil, got %+v", creds)
	}
}

func TestLoad_ReturnsCredsFromFile(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	if err := s.Save(&Credentials{APIKey: "from-file", Namespace: "ns1"}); err != nil {
		t.Fatal(err)
	}

	creds, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if creds == nil || creds.APIKey != "from-file" || creds.Namespace != "ns1" {
		t.Errorf("unexpected creds: %+v", creds)
	}
}

func TestLoad_OverlaysAPIKeyEnvVar(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	if err := s.Save(&Credentials{APIKey: "file-key"}); err != nil {
		t.Fatal(err)
	}

	t.Setenv("RISKOS_API_KEY", "env-key")
	t.Setenv("RISKOS_EVAL_KEY", "")

	creds, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if creds.APIKey != "env-key" {
		t.Errorf("expected env-key, got %s", creds.APIKey)
	}
}

func TestLoad_OverlaysEvalKeyEnvVar(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	if err := s.Save(&Credentials{APIKey: "key1"}); err != nil {
		t.Fatal(err)
	}

	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "env-eval-key")

	creds, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if creds.EvalKey != "env-eval-key" {
		t.Errorf("expected env-eval-key, got %s", creds.EvalKey)
	}
}

func TestLoad_ReturnsNilWhenAPIKeyMissing(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	// Save creds without an API key (shouldn't normally happen but test the guard)
	if err := s.Save(&Credentials{Namespace: "ns"}); err != nil {
		t.Fatal(err)
	}

	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	creds, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if creds != nil {
		t.Errorf("expected nil when api_key missing, got %+v", creds)
	}
}

func TestLoad_HandlesCorruptedJSONGracefully(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	// Write garbage JSON
	if err := os.WriteFile(s.CredFile(), []byte("{{not valid json}}"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	// Should not panic; returns nil (no API key)
	creds, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if creds != nil {
		t.Errorf("expected nil for corrupted JSON, got %+v", creds)
	}
}

func TestSave_CreatesDirectoryIfMissing(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "nested", "dir")
	s := NewStore(dir)

	if err := s.Save(&Credentials{APIKey: "key1"}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(s.CredFile()); err != nil {
		t.Errorf("expected file to exist: %v", err)
	}
}
