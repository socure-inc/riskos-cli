// Package api provides a direct REST client for the RiskOS API.
//
// Auth model:
//   - CLI/MCP key  → Authorization: Bearer <api-key>  →  /api/mcp/* endpoints
//   - Eval key     → Authorization: Bearer <eval-key> →  /api/evaluation
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/effectivai/riskos-cli/internal/config"
	"github.com/effectivai/riskos-cli/internal/debug"
)

// ErrFeatureNotAvailable indicates the connected RiskOS server does not yet
// support an endpoint (e.g. it predates a rollout). Clients should treat it as
// a soft "not available yet" condition rather than a hard error.
var ErrFeatureNotAvailable = errors.New("feature not available on the connected server yet")

const (
	DefaultBaseURL = "https://riskos.sandbox.socure.com"

	// evalMaxAttempts is the total number of attempts for RunEvaluation (initial + retries).
	evalMaxAttempts = 3
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
	base := creds.BaseURL
	if base == "" {
		base = DefaultBaseURL
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

type doOpts struct {
	debugBody string // if non-empty, logged instead of raw body in debug mode
}

func (c *RiskOSClient) do(ctx context.Context, method, path string, body io.Reader, opts ...doOpts) ([]byte, int, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, 0, fmt.Errorf("reading request body: %w", err)
		}
		body = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, 0, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	debug.Log("→ %s %s", method, c.baseURL+path)
	debug.Log("→ Headers: %s", redactHeaders(req.Header))
	if len(bodyBytes) > 0 {
		logBody := string(bodyBytes)
		if len(opts) > 0 && opts[0].debugBody != "" {
			logBody = opts[0].debugBody
		}
		debug.Log("→ Body: %s", logBody)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response body: %w", err)
	}

	debug.Log("← %d %s (%d bytes)", resp.StatusCode, resp.Status, len(b))
	debug.Log("← Headers: %s", resp.Header)
	debug.Log("← Body: %s", string(b))

	return b, resp.StatusCode, nil
}

func redactHeaders(h http.Header) string {
	var parts []string
	for k, vals := range h {
		v := strings.Join(vals, ", ")
		if strings.EqualFold(k, "Authorization") {
			v = "Bearer ***"
		}
		parts = append(parts, k+": "+v)
	}
	return strings.Join(parts, " | ")
}

func (c *RiskOSClient) getInto(ctx context.Context, path string, dest any) error {
	b, status, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", status, string(b))
	}
	if err := json.Unmarshal(b, dest); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	return nil
}

func (c *RiskOSClient) postInto(ctx context.Context, path string, payload, dest any, opts ...doOpts) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	b, status, err := c.do(ctx, http.MethodPost, path, bytes.NewReader(encoded), opts...)
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return fmt.Errorf("HTTP %d: %s", status, string(b))
	}
	if dest != nil {
		if err := json.Unmarshal(b, dest); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}

func (c *RiskOSClient) patchInto(ctx context.Context, path string, payload, dest any, opts ...doOpts) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	b, status, err := c.do(ctx, http.MethodPatch, path, bytes.NewReader(encoded), opts...)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", status, string(b))
	}
	if dest != nil {
		if err := json.Unmarshal(b, dest); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}

// ─── Auth ─────────────────────────────────────────────────────────────────────

// VerifyAPIKey calls GET /api/mcp/auth/verify with the CLI/MCP key.
func (c *RiskOSClient) VerifyAPIKey(ctx context.Context) (*AuthVerifyResult, error) {
	b, status, err := c.do(ctx, http.MethodGet, "/api/mcp/auth/verify", nil)
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

func (c *RiskOSClient) ListUseCases(ctx context.Context) ([]UseCase, error) {
	b, status, err := c.do(ctx, http.MethodGet, "/api/mcp/use-cases", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(b))
	}
	// API may return a bare array or {"useCases": [...]}
	var ucs []UseCase
	if err := json.Unmarshal(b, &ucs); err == nil {
		return ucs, nil
	}
	var wrapped useCasesResponse
	if err := json.Unmarshal(b, &wrapped); err != nil {
		return nil, fmt.Errorf("unmarshal use cases: %w", err)
	}
	return wrapped.UseCases, nil
}

// ─── Workflows ────────────────────────────────────────────────────────────────

func (c *RiskOSClient) ListWorkflows(ctx context.Context, useCase string) ([]Workflow, error) {
	b, status, err := c.do(ctx, http.MethodGet, "/api/mcp/workflows?use_case="+url.QueryEscape(useCase), nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(b))
	}
	var wfs []Workflow
	if err := json.Unmarshal(b, &wfs); err == nil {
		return wfs, nil
	}
	var wrapped workflowsResponse
	if err := json.Unmarshal(b, &wrapped); err != nil {
		return nil, fmt.Errorf("unmarshal workflows: %w", err)
	}
	return wrapped.Workflows, nil
}

// ─── Integration Checklist ────────────────────────────────────────────────────

func (c *RiskOSClient) GetChecklist(ctx context.Context, useCase, blueprint string) ([]ChecklistTopic, error) {
	path := "/api/mcp/integration-checklist?use_case=" + url.QueryEscape(useCase)
	if blueprint != "" {
		path += "&blueprint_name=" + url.QueryEscape(blueprint)
	}
	b, status, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(b))
	}
	var topics []ChecklistTopic
	if err := json.Unmarshal(b, &topics); err == nil {
		return topics, nil
	}
	var wrapped checklistResponse
	if err := json.Unmarshal(b, &wrapped); err != nil {
		return nil, fmt.Errorf("unmarshal checklist: %w", err)
	}
	return wrapped.Checklist, nil
}

// ─── Webhook Events ───────────────────────────────────────────────────────────

func (c *RiskOSClient) ListWebhookEvents(ctx context.Context) ([]WebhookEventGroup, error) {
	b, status, err := c.do(ctx, http.MethodGet, "/api/mcp/webhook-events", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(b))
	}
	var groups []WebhookEventGroup
	if err := json.Unmarshal(b, &groups); err == nil {
		return groups, nil
	}
	var wrapped webhookEventsResponse
	if err := json.Unmarshal(b, &wrapped); err != nil {
		return nil, fmt.Errorf("unmarshal webhook events: %w", err)
	}
	return wrapped.WebhookEvents, nil
}

// ─── Webhooks ─────────────────────────────────────────────────────────────────

func (c *RiskOSClient) ListWebhooks(ctx context.Context) ([]WebhookConfig, error) {
	b, status, err := c.do(ctx, http.MethodGet, "/api/mcp/webhooks", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(b))
	}
	var raw rawWebhooksResponse
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal webhooks: %w", err)
	}
	configs := make([]WebhookConfig, 0, len(raw.Configs))
	for _, r := range raw.Configs {
		useCase := ""
		if len(r.UseCaseEvents) > 0 {
			useCase = r.UseCaseEvents[0].UseCase
		}
		authScheme, _ := r.Medium.AuthSpec["authScheme"].(string)
		if authScheme == "" {
			switch {
			case r.Medium.AuthSpec["token"] != nil:
				authScheme = "BEARER_TOKEN"
			case r.Medium.AuthSpec["username"] != nil:
				authScheme = "BASIC_AUTH"
			}
		}
		configs = append(configs, WebhookConfig{
			Name:       r.Medium.Name,
			URL:        r.Medium.URL,
			UseCase:    useCase,
			AuthScheme: authScheme,
		})
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

func (c *RiskOSClient) CreateWebhook(ctx context.Context, p *WebhookParams) (map[string]any, error) {
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
	safePayload := map[string]any{
		"medium": map[string]any{
			"name":     p.Name,
			"url":      p.URL,
			"authSpec": map[string]string{"authScheme": p.AuthScheme, "credentials": "[REDACTED]"},
		},
		"useCaseEvents": []map[string]any{{
			"useCase":    p.UseCase,
			"compEvents": buildCompEvents(p),
		}},
	}
	safeBody, _ := json.Marshal(safePayload)
	var result map[string]any
	if err := c.postInto(ctx, "/api/mcp/webhooks", payload, &result, doOpts{debugBody: string(safeBody)}); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *RiskOSClient) UpdateWebhook(ctx context.Context, p *WebhookParams) (map[string]any, error) {
	payload := map[string]any{
		"url": p.URL,
		"useCaseEvents": []map[string]any{{
			"useCase":    p.UseCase,
			"compEvents": buildCompEvents(p),
		}},
	}
	var result map[string]any
	if err := c.patchInto(ctx, "/api/mcp/webhooks/"+url.QueryEscape(p.Name), payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ─── Test Cases ───────────────────────────────────────────────────────────────

// ListTestCases loads active evaluation test cases for the given use case via
// GET /api/mcp/test-cases/v2. A 404 means the connected server does not yet
// expose the v2 endpoint (it predates this rollout); that is surfaced as
// ErrFeatureNotAvailable so callers can show a soft message instead of a hard
// error.
func (c *RiskOSClient) ListTestCases(ctx context.Context, useCase string) ([]TestCase, error) {
	b, status, err := c.do(ctx, http.MethodGet, "/api/mcp/test-cases/v2?use_case="+url.QueryEscape(useCase), nil)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, ErrFeatureNotAvailable
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(b))
	}
	var tcs []TestCase
	if err := json.Unmarshal(b, &tcs); err == nil {
		return tcs, nil
	}
	var wrapped testCasesResponse
	if err := json.Unmarshal(b, &wrapped); err != nil {
		return nil, fmt.Errorf("unmarshal test cases: %w", err)
	}
	return wrapped.TestCases, nil
}

// ─── Evaluation API ───────────────────────────────────────────────────────────

// RunEvaluation calls POST /api/evaluation using the Evaluation API key (not the CLI key).
func (c *RiskOSClient) RunEvaluation(ctx context.Context, evalKey string, req EvalRequest) (*EvalResponse, error) {
	if evalKey == "" {
		return nil, fmt.Errorf("no Evaluation API key — run: riskos auth set-eval-key --api-key <uuid>")
	}
	encoded, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal eval request: %w", err)
	}
	for attempt := 0; attempt < evalMaxAttempts; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/evaluation", bytes.NewReader(encoded))
		if err != nil {
			return nil, fmt.Errorf("building eval request: %w", err)
		}
		httpReq.Header.Set("Authorization", "Bearer "+evalKey)
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json")

		debug.Log("→ POST %s (attempt %d/%d)", c.baseURL+"/api/evaluation", attempt+1, evalMaxAttempts)
		debug.Log("→ Headers: %s", redactHeaders(httpReq.Header))
		debug.Log("→ Body: %s", string(encoded))

		resp, err := c.http.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("cannot reach RiskOS API: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading eval response: %w", err)
		}

		debug.Log("← %d %s (%d bytes)", resp.StatusCode, resp.Status, len(body))
		debug.Log("← Body: %s", string(body))

		if resp.StatusCode == 429 && attempt < evalMaxAttempts-1 {
			retryAfter := evalDefaultRetryAfter
			var eb map[string]any
			if json.Unmarshal(body, &eb) == nil {
				if v, ok := eb["retry_after_seconds"].(float64); ok {
					retryAfter = int(v)
				}
			}
			fmt.Printf("\r  Rate limited — retrying in %ds (attempt %d/%d)…\n", retryAfter, attempt+1, evalMaxAttempts-1)
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
	return nil, fmt.Errorf("evaluation failed after %d attempts", evalMaxAttempts)
}
