package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/effectivai/riskos-cli/internal/api"
	"github.com/effectivai/riskos-cli/internal/config"
	"github.com/effectivai/riskos-cli/internal/ui"
)

func TestCompatibilityNotice(t *testing.T) {
	t.Setenv("RISKOS_QUIET", "")
	originalTTY := ui.IsTTY
	ui.IsTTY = true
	t.Cleanup(func() {
		ui.IsTTY = originalTTY
	})

	command := buildRootCmd(&Deps{
		Store: config.NewStore(t.TempDir()),
		ClientFn: func(*cobra.Command, *config.Credentials) api.Client {
			return &api.MockClient{}
		},
	})
	var stderr bytes.Buffer
	command.SetErr(&stderr)
	command.SetArgs([]string{"auth", "status", "--output", "plain"})

	stdout := captureStdout(t, func() {
		if err := command.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})
	if !strings.Contains(stderr.String(), deprecationNotice) {
		t.Fatalf("expected compatibility notice, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), migrationURL) {
		t.Fatalf("expected migration URL, got %q", stderr.String())
	}
	if strings.Contains(stdout, deprecationNotice) || strings.Contains(stdout, migrationURL) {
		t.Fatalf("compatibility notice must not be written to stdout, got %q", stdout)
	}
}

func TestCompatibilityNoticeQuiet(t *testing.T) {
	t.Setenv("RISKOS_QUIET", "")
	originalTTY := ui.IsTTY
	ui.IsTTY = true
	t.Cleanup(func() {
		ui.IsTTY = originalTTY
	})

	command := buildRootCmd(&Deps{
		Store: config.NewStore(t.TempDir()),
		ClientFn: func(*cobra.Command, *config.Credentials) api.Client {
			return &api.MockClient{}
		},
	})
	var stderr bytes.Buffer
	command.SetErr(&stderr)
	command.SetArgs([]string{"auth", "status", "--output", "plain", "--quiet"})

	if err := command.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Contains(stderr.String(), deprecationNotice) {
		t.Fatalf("did not expect compatibility notice in quiet mode, got %q", stderr.String())
	}
}

func TestCompatibilityNoticeEnvironmentQuiet(t *testing.T) {
	t.Setenv("RISKOS_QUIET", "1")
	originalTTY := ui.IsTTY
	ui.IsTTY = true
	t.Cleanup(func() {
		ui.IsTTY = originalTTY
	})

	command := buildRootCmd(&Deps{
		Store: config.NewStore(t.TempDir()),
		ClientFn: func(*cobra.Command, *config.Credentials) api.Client {
			return &api.MockClient{}
		},
	})
	var stderr bytes.Buffer
	command.SetErr(&stderr)
	command.SetArgs([]string{"auth", "status", "--output", "plain"})

	if err := command.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Contains(stderr.String(), deprecationNotice) {
		t.Fatalf("did not expect compatibility notice with RISKOS_QUIET=1, got %q", stderr.String())
	}
}

func TestCompatibilityNoticeJSONOutput(t *testing.T) {
	t.Setenv("RISKOS_QUIET", "")
	originalTTY := ui.IsTTY
	ui.IsTTY = true
	t.Cleanup(func() {
		ui.IsTTY = originalTTY
	})

	command := buildRootCmd(&Deps{
		Store: config.NewStore(t.TempDir()),
		ClientFn: func(*cobra.Command, *config.Credentials) api.Client {
			return &api.MockClient{}
		},
	})
	var stderr bytes.Buffer
	command.SetErr(&stderr)
	command.SetArgs([]string{"auth", "status", "--output", "json"})

	stdout := captureStdout(t, func() {
		if err := command.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})
	if strings.Contains(stderr.String(), deprecationNotice) {
		t.Fatalf("did not expect compatibility notice for JSON output, got %q", stderr.String())
	}
	if strings.Contains(stdout, deprecationNotice) || strings.Contains(stdout, migrationURL) {
		t.Fatalf("compatibility notice must not be written to stdout, got %q", stdout)
	}
}

func TestCompatibilityNoticeNonTTY(t *testing.T) {
	t.Setenv("RISKOS_QUIET", "")
	originalTTY := ui.IsTTY
	ui.IsTTY = false
	t.Cleanup(func() {
		ui.IsTTY = originalTTY
	})

	command := buildRootCmd(&Deps{
		Store: config.NewStore(t.TempDir()),
		ClientFn: func(*cobra.Command, *config.Credentials) api.Client {
			return &api.MockClient{}
		},
	})
	var stderr bytes.Buffer
	command.SetErr(&stderr)
	command.SetArgs([]string{"auth", "status", "--output", "plain"})

	if err := command.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Contains(stderr.String(), deprecationNotice) {
		t.Fatalf("did not expect compatibility notice outside a TTY, got %q", stderr.String())
	}
}
