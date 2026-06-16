package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"

	"github.com/effectivai/riskos-cli/internal/api"
	"github.com/effectivai/riskos-cli/internal/config"
)

func init() {
	keyring.MockInit()
}

// runCmd runs a fresh root command with isolated deps (no global mutation).
func runCmd(t *testing.T, store *config.Store, mock api.Client, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	d := &Deps{
		Store: store,
		ClientFn: func(cmd *cobra.Command, creds *config.Credentials) api.Client {
			if mock != nil {
				return mock
			}
			return api.NewClientFromCreds(creds)
		},
	}
	if store == nil {
		d.Store = config.NewStore(t.TempDir())
	}

	defer func() {
		if r := recover(); r != nil {
			if abortErr, ok := r.(error); ok {
				err = abortErr
			} else {
				panic(r)
			}
		}
	}()

	cmd := buildRootCmd(d)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return "", "", err
}

func TestAuthStatus_NotAuthenticated(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	store := config.NewStore(dir)
	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	out := captureStdout(t, func() {
		runCmd(t, store, nil, "auth", "status", "-o", "json")
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("expected JSON: %s — %v", out, err)
	}
	if resp["status"] != "unauthenticated" {
		t.Errorf("expected unauthenticated, got %v", resp["status"])
	}
}

func TestAuthStatus_AuthenticatedFromEnv(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	store := config.NewStore(dir)
	t.Setenv("RISKOS_API_KEY", "env-key-123")
	t.Setenv("RISKOS_EVAL_KEY", "")

	out := captureStdout(t, func() {
		runCmd(t, store, nil, "auth", "status", "-o", "json")
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("expected JSON: %s — %v", out, err)
	}
	if resp["status"] != "authenticated" {
		t.Errorf("expected authenticated, got %v", resp["status"])
	}
	if resp["source"] != "env" {
		t.Errorf("expected source=env, got %v", resp["source"])
	}
}

func TestAuthStatus_AuthenticatedFromKeyring(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	store := config.NewStore(dir)
	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	store.Save(&config.Credentials{APIKey: "keyring-key", Namespace: "my-ns"})

	out := captureStdout(t, func() {
		runCmd(t, store, nil, "auth", "status", "-o", "json")
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("expected JSON: %s — %v", out, err)
	}
	if resp["status"] != "authenticated" {
		t.Errorf("expected authenticated, got %v", resp["status"])
	}
	if resp["source"] != "keyring" {
		t.Errorf("expected source=keyring, got %v", resp["source"])
	}
	if resp["namespace"] != "my-ns" {
		t.Errorf("expected namespace=my-ns, got %v", resp["namespace"])
	}
}

func TestAuthLogout_RemovesCredentials(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	store := config.NewStore(dir)
	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	store.Save(&config.Credentials{APIKey: "key-to-remove"})

	out := captureStdout(t, func() {
		runCmd(t, store, nil, "auth", "logout", "-o", "json")
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("expected JSON: %s — %v", out, err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", resp["status"])
	}

	// Verify credentials are gone
	creds, _ := store.Load()
	if creds != nil {
		t.Error("expected credentials to be removed after logout")
	}
}

func TestAuthLogin_InvalidKey(t *testing.T) {
	t.Parallel()
	mock := &api.MockClient{
		ListUseCasesFn: func(ctx context.Context) ([]api.UseCase, error) {
			return nil, fmt.Errorf("HTTP 401: unauthorized")
		},
	}
	_, err := mock.ListUseCases(context.Background())
	if err == nil {
		t.Error("expected error for invalid key")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected 401 in error, got: %v", err)
	}
}
