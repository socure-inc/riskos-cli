// Package api provides a direct REST client for the RiskOS API.
package api

// ─── Response wrappers for direct unmarshaling ───────────────────────────────

type useCasesResponse struct {
	UseCases []UseCase `json:"useCases"`
}

type workflowsResponse struct {
	Workflows []Workflow `json:"workflows"`
}

type checklistResponse struct {
	Checklist []ChecklistTopic `json:"checklist"`
}

type webhookEventsResponse struct {
	WebhookEvents []WebhookEventGroup `json:"webhookEvents"`
}

type rawWebhookConfig struct {
	Medium struct {
		Name     string         `json:"name"`
		URL      string         `json:"url"`
		AuthSpec map[string]any `json:"authSpec"`
	} `json:"medium"`
	UseCaseEvents []struct {
		UseCase string `json:"useCase"`
	} `json:"useCaseEvents"`
}

type rawWebhooksResponse struct {
	Configs []rawWebhookConfig `json:"configs"`
	Errs    []string           `json:"errs"`
}

type testCasesResponse struct {
	TestCases []TestCase `json:"testCases"`
}

// ─── Domain types ────────────────────────────────────────────────────────────

// AuthVerifyResult is returned by VerifyAPIKey.
type AuthVerifyResult struct {
	Valid     bool   `json:"valid"`
	Namespace string `json:"namespace"`
}

// UseCase represents a RiskOS use case configured for the account.
type UseCase struct {
	Name          string `json:"name"`
	DisplayName   string `json:"displayName"`
	SolutionName  string `json:"solutionName"`
	BlueprintName string `json:"blueprintName"`
}

// Workflow represents a workflow associated with a use case.
type Workflow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Namespace   string `json:"namespace"`
	Version     string `json:"version"`
}

// ChecklistStep is a single step inside a ChecklistTopic.
type ChecklistStep struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	ContentSecondary string `json:"contentSecondary"`
	ContentType      string `json:"contentType"`
}

// ChecklistTopic groups related ChecklistSteps under a heading.
type ChecklistTopic struct {
	Topic string          `json:"topic"`
	Steps []ChecklistStep `json:"steps"`
}

// WebhookEventGroup groups event names by component (workflow, case, etc.).
type WebhookEventGroup struct {
	Component  string   `json:"component"`
	EventNames []string `json:"eventNames"`
}

// WebhookConfig describes a webhook endpoint registered for this account.
type WebhookConfig struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	UseCase    string `json:"useCase"`
	AuthScheme string `json:"authScheme"`
}

// WebhookParams holds all parameters needed to create or update a webhook.
type WebhookParams struct {
	Name, URL, UseCase             string
	WorkflowEvents, CaseEvents     []string
	AuthScheme, BearerToken        string
	BasicAuthUsername              string
	BasicAuthPassword              string
	OAuth2TokenURL, OAuth2ClientID string
	OAuth2ClientSecret             string
	OAuth2Scopes                   []string
	OAuth2AuthStyle                string
}

// TestCase is a sandbox test case returned by ListTestCases.
type TestCase struct {
	ID          string `json:"id"`
	TestName    string `json:"testName"`
	Description string `json:"description"`
	UseCase     string `json:"useCase"`
	Decision    string `json:"decision"`
	Solution    string `json:"solution"`
	IsActive    bool   `json:"isActive"`
	HTTPMethod  string `json:"httpMethod"`
	URL         string `json:"url"`
	ConfigType  string `json:"configType"`
}

// EvalRequest is the payload sent to POST /api/evaluation.
type EvalRequest struct {
	ID        string          `json:"id"`
	Timestamp string          `json:"timestamp"`
	Workflow  string          `json:"workflow"`
	Data      EvalRequestData `json:"data"`
}

// EvalRequestData carries the individual and custom fields for an evaluation.
type EvalRequestData struct {
	Individual map[string]string `json:"individual"`
	Custom     map[string]string `json:"custom"`
}

// EvalResponse is returned by the evaluation API after a successful POST.
type EvalResponse struct {
	ID          string `json:"id"`
	Workflow    string `json:"workflow"`
	WorkflowID  string `json:"workflow_id"`
	WorkflowVer string `json:"workflow_version"`
	EvalSource  string `json:"eval_source"`
	EvalID      string `json:"eval_id"`
	Decision    string `json:"decision"`
	EvalStatus  string `json:"eval_status"`
	Environment string `json:"environment_name"`
	RedirectURI string `json:"redirect_uri,omitempty"`
}
