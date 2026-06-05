package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"riskos/internal/api"
	"riskos/internal/config"
)

func TestUsecasesList_JSON(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStore(dir)
	store.Save(&config.Credentials{APIKey: "key"})
	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	mock := &api.MockClient{
		ListUseCasesFn: func() ([]api.UseCase, error) {
			return []api.UseCase{
				{Name: "consumer_onboarding", DisplayName: "Consumer Onboarding", BlueprintName: "bp1"},
			}, nil
		},
	}

	out := captureStdout(t, func() {
		runCmd(store, mock, "usecases", "list", "-o", "json")
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("expected JSON, got: %s — err: %v", out, err)
	}
	ucs, ok := resp["useCases"].([]interface{})
	if !ok || len(ucs) == 0 {
		t.Errorf("expected useCases array, got: %v", resp)
	}
}

func TestUsecasesList_Plain(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStore(dir)
	store.Save(&config.Credentials{APIKey: "key"})
	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	mock := &api.MockClient{
		ListUseCasesFn: func() ([]api.UseCase, error) {
			return []api.UseCase{
				{Name: "consumer_onboarding", DisplayName: "Consumer Onboarding"},
			}, nil
		},
	}

	out := captureStdout(t, func() {
		runCmd(store, mock, "usecases", "list", "-o", "plain")
	})
	if !strings.Contains(out, "consumer_onboarding") {
		t.Errorf("expected use case name in output, got: %s", out)
	}
}

func TestUsecasesList_ServerError(t *testing.T) {
	mock := &api.MockClient{
		ListUseCasesFn: func() ([]api.UseCase, error) {
			return nil, fmt.Errorf("HTTP 500: internal server error")
		},
	}

	_, err := mock.ListUseCases()
	if err == nil {
		t.Error("expected error from server error mock")
	}
}
