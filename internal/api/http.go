// Package api provides a direct REST client for the RiskOS API.
//
// Auth model:
//   - CLI/MCP key  → Authorization: Bearer <api-key>  →  /api/mcp/* endpoints
//   - Eval key     → Authorization: Bearer <eval-key> →  /api/evaluation
//
// Base URLs:
//   - Sandbox:    https://riskos.sandbox.socure.com
//   - Production: https://riskos.socure.com
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"riskos/internal/config"
)

const (
	BaseURLSandbox = "https://riskos.sandbox.socure.com"
	BaseURLProd    = "https://riskos.socure.com"

	// evalMaxRetries is the number of retry attempts for RunEvaluation.
	evalMaxRetries = 2
	// evalDefaultRetryAfter is the fallback retry delay in seconds when the
	// server does not provide a Retry-After value.
	evalDefaultRetryAfter = 2
)

// RiskOSClient is the production implementation of Client.
type RiskOSClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewClientFromCreds builds a RiskOSClient from stored credentials.
func NewClientFromCreds(creds *config.Credentials) *RiskOSClient {
	base := BaseURLSandbox
	if creds.Prod {
		base = BaseURLProd
	}
	return &RiskOSClient{
		baseURL: base,
		apiKey:  creds.APIKey,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

// NewTestClient builds a RiskOSClient with an explicit base URL — useful in tests.
func NewTestClient(apiKey, baseURL string) *RiskOSClient {
	return &RiskOSClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

// ─── core HTTP ────────────────────────────────────────────────────────────────

func (c *RiskOSClient) do(ctx context.Context, method, path string, body io.Reader) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, 0, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response body: %w", err)
	}
	return b, resp.StatusCode, nil
}

func (c *RiskOSClient) get(path string) (map[string]any, error) {
	b, status, err := c.do(context.Background(), http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(b))
	}
	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return result, nil
}

func (c *RiskOSClient) post(path string, payload any) (map[string]any, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	b, status, err := c.do(context.Background(), http.MethodPost, path, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(b))
	}
	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return result, nil
}

func (c *RiskOSClient) patch(path string, payload any) (map[string]any, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	b, status, err := c.do(context.Background(), http.MethodPatch, path, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(b))
	}
	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return result, nil
}

// ─── Auth ─────────────────────────────────────────────────────────────────────

// VerifyAPIKey calls GET /api/mcp/auth/verify with the CLI/MCP key.
func (c *RiskOSClient) VerifyAPIKey() (*AuthVerifyResult, error) {
	b, status, err := c.do(context.Background(), http.MethodGet, "/api/mcp/auth/verify", nil)
	if err != nil {
		return nil, err
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return nil, fmt.Errorf("invalid API key (HTTP %d)", status)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("auth verify failed (HTTP %d): %s", status, string(b))
	}
	var result AuthVerifyResult
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("unmarshal auth verify: %w", err)
	}
	if !result.Valid {
		return nil, fmt.Errorf("API key is not valid")
	}
	return &result, nil
}

// ─── Use Cases ────────────────────────────────────────────────────────────────

func (c *RiskOSClient) ListUseCases() ([]UseCase, error) {
	result, err := c.get("/api/mcp/use-cases")
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(result["useCases"])
	if err != nil {
		return nil, fmt.Errorf("marshal useCases: %w", err)
	}
	var ucs []UseCase
	if err := json.Unmarshal(b, &ucs); err != nil {
		return nil, fmt.Errorf("unmarshal useCases: %w", err)
	}
	return ucs, nil
}

// ─── Workflows ────────────────────────────────────────────────────────────────

func (c *RiskOSClient) ListWorkflows(useCase string) ([]Workflow, error) {
	result, err := c.get("/api/mcp/workflows?use_case=" + url.QueryEscape(useCase))
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(result["workflows"])
	if err != nil {
		return nil, fmt.Errorf("marshal workflows: %w", err)
	}
	var wfs []Workflow
	if err := json.Unmarshal(b, &wfs); err != nil {
		return nil, fmt.Errorf("unmarshal workflows: %w", err)
	}
	return wfs, nil
}

// ─── Integration Checklist ────────────────────────────────────────────────────

func (c *RiskOSClient) GetChecklist(useCase, blueprint string) ([]ChecklistTopic, error) {
	path := "/api/mcp/integration-checklist?use_case=" + url.QueryEscape(useCase)
	if blueprint != "" {
		path += "&blueprint_name=" + url.QueryEscape(blueprint)
	}
	result, err := c.get(path)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(result["checklist"])
	if err != nil {
		return nil, fmt.Errorf("marshal checklist: %w", err)
	}
	var topics []ChecklistTopic
	if err := json.Unmarshal(b, &topics); err != nil {
		return nil, fmt.Errorf("unmarshal checklist: %w", err)
	}
	return topics, nil
}

// ─── Webhook Events ───────────────────────────────────────────────────────────

func (c *RiskOSClient) ListWebhookEvents() ([]WebhookEventGroup, error) {
	result, err := c.get("/api/mcp/webhook-events")
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(result["webhookEvents"])
	if err != nil {
		return nil, fmt.Errorf("marshal webhookEvents: %w", err)
	}
	var groups []WebhookEventGroup
	if err := json.Unmarshal(b, &groups); err != nil {
		return nil, fmt.Errorf("unmarshal webhookEvents: %w", err)
	}
	return groups, nil
}

// ─── Webhooks ─────────────────────────────────────────────────────────────────

func (c *RiskOSClient) ListWebhooks() ([]WebhookConfig, error) {
	result, err := c.get("/api/mcp/webhooks")
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(result["webhooks"])
	if err != nil {
		return nil, fmt.Errorf("marshal webhooks: %w", err)
	}
	var configs []WebhookConfig
	if err := json.Unmarshal(b, &configs); err != nil {
		return nil, fmt.Errorf("unmarshal webhooks: %w", err)
	}
	return configs, nil
}

func buildAuthSpec(p *WebhookParams) map[string]any {
	switch p.AuthScheme {
	case "BEARER_TOKEN":
		return map[string]any{
			"authScheme":  "BEARER_TOKEN",
			"bearerToken": map[string]string{"token": p.BearerToken},
		}
	case "BASIC_AUTH":
		return map[string]any{
			"authScheme": "BASIC_AUTH",
			"basicAuth":  map[string]string{"username": p.BasicAuthUsername, "password": p.BasicAuthPassword},
		}
	case "OAUTH2":
		return map[string]any{
			"authScheme": "OAUTH2",
			"oauth2": map[string]any{
				"tokenUrl": p.OAuth2TokenURL, "clientId": p.OAuth2ClientID,
				"clientSecret": p.OAuth2ClientSecret, "scopes": p.OAuth2Scopes,
				"authStyle": p.OAuth2AuthStyle,
			},
		}
	}
	return nil
}

func buildCompEvents(p *WebhookParams) []map[string]any {
	var events []map[string]any
	if len(p.CaseEvents) > 0 {
		events = append(events, map[string]any{
			"component":    "case",
			"eventFormats": map[string]any{"names": p.CaseEvents},
		})
	}
	if len(p.WorkflowEvents) > 0 {
		events = append(events, map[string]any{
			"component":    "workflow",
			"eventFormats": map[string]any{"names": p.WorkflowEvents},
		})
	}
	return events
}

func (c *RiskOSClient) CreateWebhook(p *WebhookParams) (map[string]any, error) {
	payload := map[string]any{
		"medium": map[string]any{
			"name":     p.Name,
			"url":      p.URL,
			"authSpec": buildAuthSpec(p),
		},
		"useCaseEvents": []map[string]any{{
			"useCase":    p.UseCase,
			"compEvents": buildCompEvents(p),
		}},
	}
	return c.post("/api/mcp/webhooks", payload)
}

func (c *RiskOSClient) UpdateWebhook(p *WebhookParams) (map[string]any, error) {
	payload := map[string]any{
		"url": p.URL,
		"useCaseEvents": []map[string]any{{
			"useCase":    p.UseCase,
			"compEvents": buildCompEvents(p),
		}},
	}
	return c.patch("/api/mcp/webhooks/"+url.QueryEscape(p.Name), payload)
}

// ─── Test Cases ───────────────────────────────────────────────────────────────

func (c *RiskOSClient) ListTestCases(blueprint string) ([]TestCase, error) {
	result, err := c.get("/api/mcp/test-cases?blueprint_name=" + url.QueryEscape(blueprint))
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(result["testCases"])
	if err != nil {
		return nil, fmt.Errorf("marshal testCases: %w", err)
	}
	var tcs []TestCase
	if err := json.Unmarshal(b, &tcs); err != nil {
		return nil, fmt.Errorf("unmarshal testCases: %w", err)
	}
	return tcs, nil
}

// ─── Evaluation API ───────────────────────────────────────────────────────────

// RunEvaluation calls POST /api/evaluation using the Evaluation API key (not the CLI key).
func (c *RiskOSClient) RunEvaluation(evalKey string, req EvalRequest) (*EvalResponse, error) {
	if evalKey == "" {
		return nil, fmt.Errorf("no Evaluation API key — run: riskos auth set-eval-key --api-key <uuid>")
	}
	encoded, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal eval request: %w", err)
	}
	for attempt := 0; attempt < evalMaxRetries; attempt++ {
		httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, c.baseURL+"/api/evaluation", bytes.NewReader(encoded))
		if err != nil {
			return nil, fmt.Errorf("building eval request: %w", err)
		}
		httpReq.Header.Set("Authorization", "Bearer "+evalKey)
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json")

		resp, err := c.http.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("cannot reach RiskOS sandbox: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading eval response: %w", err)
		}

		if resp.StatusCode == 429 && attempt == 0 {
			retryAfter := evalDefaultRetryAfter
			var eb map[string]any
			if json.Unmarshal(body, &eb) == nil {
				if v, ok := eb["retry_after_seconds"].(float64); ok {
					retryAfter = int(v)
				}
			}
			fmt.Printf("\r  Rate limited — retrying in %ds…\n", retryAfter)
			time.Sleep(time.Duration(retryAfter+1) * time.Second)
			continue
		}
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("evaluation API %d: %s", resp.StatusCode, string(body))
		}
		var result EvalResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("unmarshal eval response: %w", err)
		}
		return &result, nil
	}
	return nil, fmt.Errorf("evaluation failed after retries")
}
