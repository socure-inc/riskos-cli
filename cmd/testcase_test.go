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
		ListTestCasesFn: func(ctx context.Context, useCase, configType string) ([]api.TestCase, error) {
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
		ListTestCasesFn: func(ctx context.Context, useCase, configType string) ([]api.TestCase, error) {
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

func TestTestcaseList_ConfigTypeMOCK(t *testing.T) {
	store := makeAuthStore(t)
	var gotConfigType string
	mock := &api.MockClient{
		ListTestCasesFn: func(ctx context.Context, useCase, configType string) ([]api.TestCase, error) {
			gotConfigType = configType
			return []api.TestCase{
				{TestName: "MOCK_Case", UseCase: useCase, Decision: "ACCEPT", ConfigType: "MOCK"},
			}, nil
		},
	}

	out := captureStdout(t, func() {
		runCmd(t, store, mock, "testcase", "list", "consumer_onboarding", "--config-type", "MOCK", "-o", "json")
	})

	if gotConfigType != "MOCK" {
		t.Errorf("expected ListTestCases configType=MOCK, got %q", gotConfigType)
	}
	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("expected JSON: %s — %v", out, err)
	}
	if resp["configType"] != "MOCK" {
		t.Errorf("expected configType MOCK in output, got %v", resp["configType"])
	}
}

func TestTestcaseFields_JSON(t *testing.T) {
	store := makeAuthStore(t)
	mock := &api.MockClient{
		ListTestCaseFieldsFn: func(ctx context.Context, useCase, workflowName, workflowVersion string) (*api.DeterministicGenerateFields, error) {
			if useCase != "consumer_onboarding" || workflowName != "consumer_onboarding" || workflowVersion != "12.0.0" {
				t.Errorf("unexpected args: %s %s %s", useCase, workflowName, workflowVersion)
			}
			return &api.DeterministicGenerateFields{
				RequiredCount:    1,
				RecommendedCount: 1,
				OptionalCount:    0,
				Fields: []api.DeterministicGenerateField{
					{Key: "individual.firstName", Requirement: "REQUIRED", Locked: true},
					{Key: "individual.email", Requirement: "RECOMMENDED"},
				},
			}, nil
		},
	}

	out := captureStdout(t, func() {
		runCmd(t, store, mock, "testcase", "fields", "consumer_onboarding", "consumer_onboarding", "12.0.0", "-o", "json")
	})

	var resp api.DeterministicGenerateFields
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("expected JSON: %s — %v", out, err)
	}
	if resp.RequiredCount != 1 || len(resp.Fields) != 2 {
		t.Errorf("unexpected fields response: %+v", resp)
	}
}

func TestTestcaseGenerate_JSON(t *testing.T) {
	store := makeAuthStore(t)
	mock := &api.MockClient{
		GenerateTestCasesFn: func(ctx context.Context, useCase, workflowName, workflowVersion string) (*api.GenerateTestCasesResult, error) {
			return &api.GenerateTestCasesResult{
				TotalPathsFound:    3,
				Verified:           2,
				ImportedCount:      2,
				DeletedCount:       1,
				ImportErrorSamples: []string{},
			}, nil
		},
	}

	out := captureStdout(t, func() {
		runCmd(t, store, mock, "testcase", "generate", "consumer_onboarding", "consumer_onboarding", "12.0.0", "-o", "json")
	})

	var resp api.GenerateTestCasesResult
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("expected JSON: %s — %v", out, err)
	}
	if resp.ImportedCount != 2 || resp.Verified != 2 {
		t.Errorf("unexpected generate response: %+v", resp)
	}
}

func TestNormalizeTestCaseConfigType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"", "", true},
		{"eval", "EVAL", true},
		{"MOCK", "MOCK", true},
		{"mock", "MOCK", true},
		{"live", "", false},
	}
	for _, tc := range cases {
		got, err := normalizeTestCaseConfigType(tc.in)
		if tc.ok {
			if err != nil || got != tc.want {
				t.Errorf("normalize(%q)=(%q,%v) want (%q,nil)", tc.in, got, err, tc.want)
			}
			continue
		}
		if err == nil {
			t.Errorf("normalize(%q) expected error", tc.in)
		}
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
