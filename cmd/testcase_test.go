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

func testcaseMock() *api.MockClient {
	return &api.MockClient{
		ListUseCasesFn: func(ctx context.Context) ([]api.UseCase, error) {
			return []api.UseCase{{Name: "consumer_onboarding", BlueprintName: "bp1"}}, nil
		},
		ListTestCasesFn: func(ctx context.Context, blueprint string) ([]api.TestCase, error) {
			return []api.TestCase{
				{TestName: "Valid_Consumer_Accept", UseCase: "consumer_onboarding", Decision: "ACCEPT", Description: "passes KYC"},
				{TestName: "Watchlist_Hit_Reject", UseCase: "consumer_onboarding", Decision: "REJECT", Description: "OFAC hit"},
			}, nil
		},
	}
}

func TestTestcaseList_UsesLiveAPI(t *testing.T) {
	store := makeAuthStore(t)

	out := captureStdout(t, func() {
		runCmd(t, store, testcaseMock(), "testcase", "list", "consumer_onboarding", "-o", "json")
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("expected JSON: %s — %v", out, err)
	}
	tcs, _ := resp["testCases"].([]interface{})
	if len(tcs) != 2 {
		t.Errorf("expected 2 test cases, got %d", len(tcs))
	}
}

func TestTestcaseList_EmptyRendersJSONArray(t *testing.T) {
	store := makeAuthStore(t)
	mock := &api.MockClient{
		ListUseCasesFn: func(ctx context.Context) ([]api.UseCase, error) {
			return []api.UseCase{{Name: "consumer_onboarding", BlueprintName: "bp1"}}, nil
		},
		ListTestCasesFn: func(ctx context.Context, blueprint string) ([]api.TestCase, error) {
			return nil, nil
		},
	}

	out := captureStdout(t, func() {
		runCmd(t, store, mock, "testcase", "list", "consumer_onboarding", "-o", "json")
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("expected JSON: %s — %v", out, err)
	}
	if !strings.Contains(out, `"testCases": []`) {
		t.Errorf("expected empty JSON array for testCases, got: %s", out)
	}
	if c, _ := resp["count"].(float64); c != 0 {
		t.Errorf("expected count 0, got %v", resp["count"])
	}
}

func TestTestcaseDescribe_WatchlistReject(t *testing.T) {
	store := makeAuthStore(t)

	out := captureStdout(t, func() {
		runCmd(t, store, testcaseMock(), "testcase", "describe", "consumer_onboarding", "Watchlist_Hit_Reject")
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
