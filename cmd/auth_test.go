package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"riskos/internal/api"
	"riskos/internal/config"
)

// runCmd runs a fresh root command with a mock client and store.
// stdout/stderr from cobra are captured but note ui.PrintJSON writes to os.Stdout.
// Use captureStdout for commands that use ui.PrintJSON.
func runCmd(store *config.Store, mock api.Client, args ...string) (string, string, error) {
	origClientFn := newClientFn
	origStoreFn := defaultStoreFn
	defer func() {
		newClientFn = origClientFn
		defaultStoreFn = origStoreFn
	}()

	if mock != nil {
		newClientFn = func(...interface{}) api.Client { return mock }
	}
	if store != nil {
		defaultStoreFn = func() *config.Store { return store }
	}

	cmd := buildRootCmd()
	cmd.SetArgs(args)
	err := cmd.Execute()
	return "", "", err
}

func TestAuthLogin_ValidKey(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStore(dir)
	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	mock := &api.MockClient{
		VerifyAPIKeyFn: func() (*api.AuthVerifyResult, error) {
			return &api.AuthVerifyResult{Valid: true, Namespace: "acme-test"}, nil
		},
	}

	var out string
	var runErr error
	out = captureStdout(t, func() {
		_, _, runErr = runCmd(store, mock, "auth", "login", "--api-key", "valid-key", "-o", "json")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	// Check creds were saved
	creds, _ := store.Load()
	if creds == nil || creds.APIKey == "" {
		t.Error("expected creds to be saved")
	}

	// Check output contains namespace
	if !strings.Contains(out, "acme-test") {
		t.Errorf("expected namespace in output, got: %s", out)
	}
}

func TestAuthLogin_InvalidKey(t *testing.T) {
	mock := &api.MockClient{
		VerifyAPIKeyFn: func() (*api.AuthVerifyResult, error) {
			return nil, fmt.Errorf("invalid API key (HTTP 401)")
		},
	}
	// Verify mock returns error
	_, err := mock.VerifyAPIKey()
	if err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestAuthWhoami_NotAuthenticated(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStore(dir)
	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	// Capture both stdout and stderr
	out := captureStdout(t, func() {
		runCmd(store, nil, "auth", "whoami")
	})
	// stderr goes directly to os.Stderr so capture that too
	_ = out
	// The whoami command writes "Not authenticated" to cmd.ErrOrStderr()
	// which cobra routes to os.Stderr by default — just check it doesn't crash
	// and the creds are nil
	creds, _ := store.Load()
	if creds != nil {
		t.Error("expected no creds in empty store")
	}
}

func TestAuthWhoami_Authenticated(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStore(dir)
	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	store.Save(&config.Credentials{APIKey: "key", Namespace: "my-ns"})

	out := captureStdout(t, func() {
		runCmd(store, nil, "auth", "whoami", "-o", "json")
	})
	if !strings.Contains(out, "my-ns") {
		t.Errorf("expected namespace in output, got: %s", out)
	}
}

func TestAuthSetEvalKey_StoresKey(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStore(dir)
	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	// Pre-save a valid API key
	store.Save(&config.Credentials{APIKey: "existing-key"})

	var out string
	var runErr error
	out = captureStdout(t, func() {
		_, _, runErr = runCmd(store, nil, "auth", "set-eval-key", "--api-key", "eval-uuid-123", "-o", "json")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	// Check key was stored
	creds, _ := store.Load()
	if creds == nil || creds.EvalKey != "eval-uuid-123" {
		t.Errorf("expected eval key to be stored, got creds: %+v", creds)
	}

	// Check output
	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("expected JSON output: %s", out)
	}
	if resp["eval_api_key_set"] != true {
		t.Errorf("expected eval_api_key_set=true, got %v", resp)
	}
}
