package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/effectivai/riskos-cli/internal/api"
	"github.com/effectivai/riskos-cli/internal/config"
	"github.com/effectivai/riskos-cli/internal/ui"
)

func makeAuthStore(t *testing.T) *config.Store {
	t.Helper()
	dir := t.TempDir()
	store := config.NewStore(dir)
	store.Save(&config.Credentials{APIKey: "key", EvalKey: "eval-key"})
	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")
	return store
}

func TestTestcaseList_ShowsAllPersonas(t *testing.T) {
	store := makeAuthStore(t)

	out := captureStdout(t, func() {
		runCmd(t, store, nil, "testcase", "list", "-o", "json")
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("expected JSON: %s — %v", out, err)
	}
	tcs, _ := resp["testCases"].([]interface{})
	if len(tcs) != 4 {
		t.Errorf("expected 4 test cases, got %d", len(tcs))
	}
}

func TestTestcaseDescribe_WatchlistReject(t *testing.T) {
	store := makeAuthStore(t)

	out := captureStdout(t, func() {
		runCmd(t, store, nil, "testcase", "describe", "Watchlist_Hit_Reject")
	})
	if !strings.Contains(out, "REJECT") {
		t.Errorf("expected REJECT in output, got: %s", out)
	}
}

func TestTestcaseRun_ValidConsumerAccept(t *testing.T) {
	store := makeAuthStore(t)

	mock := &api.MockClient{
		RunEvaluationFn: func(ctx context.Context, evalKey string, req api.EvalRequest) (*api.EvalResponse, error) {
			return &api.EvalResponse{
				EvalID:     "eval-abc",
				Decision:   "REVIEW",
				EvalStatus: "evaluation_paused",
			}, nil
		},
	}

	out := captureStdout(t, func() {
		runCmd(t, store, mock, "testcase", "run", "Valid_Consumer_Accept", "-o", "json")
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("expected JSON: %s — %v", out, err)
	}

	evalResp, _ := resp["evalResponse"].(map[string]any)
	if evalResp["eval_id"] != "eval-abc" {
		t.Errorf("expected eval_id=eval-abc, got: %v", evalResp)
	}
}

func TestTestcaseRun_WrongCase_SuggestsCorrect(t *testing.T) {
	t.Parallel()
	suggestion := ui.SuggestTypo("valid_consumer_accept", testPersonaOrder)
	if suggestion != "Valid_Consumer_Accept" {
		t.Errorf("expected suggestion Valid_Consumer_Accept, got %q", suggestion)
	}
}

func TestTestcaseRun_NoEvalKey_Error(t *testing.T) {
	t.Parallel()
	mock := &api.MockClient{
		RunEvaluationFn: func(ctx context.Context, evalKey string, req api.EvalRequest) (*api.EvalResponse, error) {
			if evalKey == "" {
				return nil, fmt.Errorf("no Evaluation API key — run: riskos auth set-eval-key --api-key <uuid>")
			}
			return nil, nil
		},
	}

	_, err := mock.RunEvaluation(context.Background(), "", api.EvalRequest{})
	if err == nil {
		t.Error("expected error when eval key missing")
	}
	if !strings.Contains(err.Error(), "set-eval-key") {
		t.Errorf("expected set-eval-key hint in error, got: %v", err)
	}
}
