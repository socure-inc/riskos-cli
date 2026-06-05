package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(handler http.HandlerFunc) (*httptest.Server, *RiskOSClient) {
	srv := httptest.NewServer(handler)
	client := NewTestClient("test-key", srv.URL)
	return srv, client
}

// ─── VerifyAPIKey ─────────────────────────────────────────────────────────────

func TestVerifyAPIKey_Valid(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/mcp/auth/verify" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"valid": true, "namespace": "acme"})
	})
	defer srv.Close()

	res, err := client.VerifyAPIKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Valid || res.Namespace != "acme" {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestVerifyAPIKey_Unauthorized(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer srv.Close()

	_, err := client.VerifyAPIKey()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifyAPIKey_InvalidJSON(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not-json"))
	})
	defer srv.Close()

	_, err := client.VerifyAPIKey()
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

// ─── ListUseCases ─────────────────────────────────────────────────────────────

func TestListUseCases_HappyPath(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"useCases": []map[string]any{
				{"name": "consumer_onboarding", "blueprintName": "bp1"},
			},
		})
	})
	defer srv.Close()

	ucs, err := client.ListUseCases()
	if err != nil {
		t.Fatal(err)
	}
	if len(ucs) != 1 || ucs[0].Name != "consumer_onboarding" {
		t.Errorf("unexpected result: %+v", ucs)
	}
}

func TestListUseCases_ServerError(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	})
	defer srv.Close()

	_, err := client.ListUseCases()
	if err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestListUseCases_Empty(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"useCases": []any{}})
	})
	defer srv.Close()

	ucs, err := client.ListUseCases()
	if err != nil {
		t.Fatal(err)
	}
	if len(ucs) != 0 {
		t.Errorf("expected empty slice, got %v", ucs)
	}
}

// ─── ListWorkflows ────────────────────────────────────────────────────────────

func TestListWorkflows_HappyPath(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "use_case=consumer_onboarding") {
			t.Errorf("expected use_case query param, got %s", r.URL.RawQuery)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"workflows": []map[string]any{
				{"id": "wf1", "name": "wf_one", "displayName": "Workflow One"},
			},
		})
	})
	defer srv.Close()

	wfs, err := client.ListWorkflows("consumer_onboarding")
	if err != nil {
		t.Fatal(err)
	}
	if len(wfs) != 1 || wfs[0].ID != "wf1" {
		t.Errorf("unexpected result: %+v", wfs)
	}
}

// ─── GetChecklist ─────────────────────────────────────────────────────────────

func TestGetChecklist_WithBlueprint(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "blueprint_name=bp1") {
			t.Errorf("expected blueprint_name, got %s", r.URL.RawQuery)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"checklist": []map[string]any{
				{"topic": "Auth", "steps": []map[string]any{{"id": "1", "title": "Step 1"}}},
			},
		})
	})
	defer srv.Close()

	topics, err := client.GetChecklist("consumer_onboarding", "bp1")
	if err != nil {
		t.Fatal(err)
	}
	if len(topics) != 1 {
		t.Errorf("expected 1 topic, got %d", len(topics))
	}
}

func TestGetChecklist_WithoutBlueprint(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.RawQuery, "blueprint_name") {
			t.Error("should not include blueprint_name when empty")
		}
		json.NewEncoder(w).Encode(map[string]any{"checklist": []any{}})
	})
	defer srv.Close()

	topics, err := client.GetChecklist("consumer_onboarding", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(topics) != 0 {
		t.Errorf("expected empty, got %v", topics)
	}
}

// ─── ListWebhookEvents ────────────────────────────────────────────────────────

func TestListWebhookEvents_HappyPath(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"webhookEvents": []map[string]any{
				{"component": "workflow", "eventNames": []string{"evaluation_completed"}},
			},
		})
	})
	defer srv.Close()

	groups, err := client.ListWebhookEvents()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Component != "workflow" {
		t.Errorf("unexpected: %+v", groups)
	}
}

// ─── ListWebhooks ─────────────────────────────────────────────────────────────

func TestListWebhooks_HappyPath(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"webhooks": []map[string]any{
				{"name": "my-hook", "url": "https://example.com", "useCase": "consumer_onboarding", "authScheme": "BEARER_TOKEN"},
			},
		})
	})
	defer srv.Close()

	configs, err := client.ListWebhooks()
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 1 || configs[0].Name != "my-hook" {
		t.Errorf("unexpected: %+v", configs)
	}
}

func TestListWebhooks_Empty(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"webhooks": []any{}})
	})
	defer srv.Close()

	configs, err := client.ListWebhooks()
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 0 {
		t.Errorf("expected empty")
	}
}

// ─── CreateWebhook ────────────────────────────────────────────────────────────

func TestCreateWebhook_BearerToken(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		medium, _ := body["medium"].(map[string]any)
		authSpec, _ := medium["authSpec"].(map[string]any)
		if authSpec["authScheme"] != "BEARER_TOKEN" {
			t.Errorf("expected BEARER_TOKEN authScheme, got %v", authSpec["authScheme"])
		}
		bearerToken, _ := authSpec["bearerToken"].(map[string]any)
		if bearerToken["token"] != "secret" {
			t.Errorf("expected token=secret, got %v", bearerToken["token"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"status": "SUCCESS"})
	})
	defer srv.Close()

	_, err := client.CreateWebhook(&WebhookParams{
		Name:        "my-hook",
		URL:         "https://example.com",
		UseCase:     "consumer_onboarding",
		AuthScheme:  "BEARER_TOKEN",
		BearerToken: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateWebhook_BasicAuth(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		medium, _ := body["medium"].(map[string]any)
		authSpec, _ := medium["authSpec"].(map[string]any)
		if authSpec["authScheme"] != "BASIC_AUTH" {
			t.Errorf("expected BASIC_AUTH, got %v", authSpec["authScheme"])
		}
		basicAuth, _ := authSpec["basicAuth"].(map[string]any)
		if basicAuth["username"] != "user" || basicAuth["password"] != "pass" {
			t.Errorf("unexpected basicAuth: %v", basicAuth)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"status": "SUCCESS"})
	})
	defer srv.Close()

	_, err := client.CreateWebhook(&WebhookParams{
		Name:              "my-hook",
		URL:               "https://example.com",
		UseCase:           "consumer_onboarding",
		AuthScheme:        "BASIC_AUTH",
		BasicAuthUsername: "user",
		BasicAuthPassword: "pass",
	})
	if err != nil {
		t.Fatal(err)
	}
}

// ─── UpdateWebhook ────────────────────────────────────────────────────────────

func TestUpdateWebhook(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "my-hook") {
			t.Errorf("expected path to contain webhook name, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"status": "SUCCESS"})
	})
	defer srv.Close()

	_, err := client.UpdateWebhook(&WebhookParams{
		Name:    "my-hook",
		URL:     "https://example.com",
		UseCase: "consumer_onboarding",
	})
	if err != nil {
		t.Fatal(err)
	}
}

// ─── ListTestCases ────────────────────────────────────────────────────────────

func TestListTestCases_HappyPath(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"testCases": []map[string]any{
				{"id": "tc1", "testName": "Valid_Consumer_Accept", "decision": "ACCEPT"},
			},
		})
	})
	defer srv.Close()

	tcs, err := client.ListTestCases("bp1")
	if err != nil {
		t.Fatal(err)
	}
	if len(tcs) != 1 || tcs[0].TestName != "Valid_Consumer_Accept" {
		t.Errorf("unexpected: %+v", tcs)
	}
}

func TestListTestCases_Empty(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"testCases": []any{}})
	})
	defer srv.Close()

	tcs, err := client.ListTestCases("bp1")
	if err != nil {
		t.Fatal(err)
	}
	if len(tcs) != 0 {
		t.Errorf("expected empty")
	}
}

// ─── RunEvaluation ────────────────────────────────────────────────────────────

func TestRunEvaluation_Success(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(EvalResponse{
			EvalID:     "eval-123",
			Decision:   "REVIEW",
			EvalStatus: "evaluation_paused",
		})
	})
	defer srv.Close()

	req := EvalRequest{ID: "req-1", Workflow: "consumer_onboarding"}
	resp, err := client.RunEvaluation("eval-key", req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.EvalID != "eval-123" {
		t.Errorf("unexpected eval_id: %s", resp.EvalID)
	}
}

func TestRunEvaluation_RateLimitRetry(t *testing.T) {
	calls := 0
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]any{"retry_after_seconds": 0})
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(EvalResponse{EvalID: "eval-retry"})
	})
	defer srv.Close()

	req := EvalRequest{ID: "req-2", Workflow: "consumer_onboarding"}
	resp, err := client.RunEvaluation("eval-key", req)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
	if resp.EvalID != "eval-retry" {
		t.Errorf("unexpected eval_id: %s", resp.EvalID)
	}
}

func TestRunEvaluation_NoEvalKey(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {})
	defer srv.Close()

	_, err := client.RunEvaluation("", EvalRequest{})
	if err == nil {
		t.Fatal("expected error for empty eval key")
	}
}

func TestRunEvaluation_NonCreatedError(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
	})
	defer srv.Close()

	_, err := client.RunEvaluation("eval-key", EvalRequest{ID: "req-3"})
	if err == nil {
		t.Fatal("expected error on 400")
	}
}
