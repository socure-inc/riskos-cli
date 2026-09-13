package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/effectivai/riskos-cli/internal/api"
	"github.com/effectivai/riskos-cli/internal/config"
)

func TestCompatibilityNotice(t *testing.T) {
	command := buildRootCmd(&Deps{
		Store: config.NewStore(t.TempDir()),
		ClientFn: func(*cobra.Command, *config.Credentials) api.Client {
			return &api.MockClient{}
		},
	})
	var stderr bytes.Buffer
	command.SetErr(&stderr)
	command.SetArgs([]string{"auth", "status", "--output", "json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(stderr.String(), deprecationNotice) {
		t.Fatalf("expected compatibility notice, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), agentKitURL) {
		t.Fatalf("expected agent kit URL, got %q", stderr.String())
	}
}

func TestCompatibilityNoticeQuiet(t *testing.T) {
	command := buildRootCmd(&Deps{
		Store: config.NewStore(t.TempDir()),
		ClientFn: func(*cobra.Command, *config.Credentials) api.Client {
			return &api.MockClient{}
		},
	})
	var stderr bytes.Buffer
	command.SetErr(&stderr)
	command.SetArgs([]string{"auth", "status", "--output", "json", "--quiet"})

	if err := command.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Contains(stderr.String(), deprecationNotice) {
		t.Fatalf("did not expect compatibility notice in quiet mode, got %q", stderr.String())
	}
}
