package api

import "context"

// Client is the interface for all RiskOS API operations.
// It is satisfied by *RiskOSClient (production) and MockClient (tests).
type Client interface {
	VerifyAPIKey(ctx context.Context) (*AuthVerifyResult, error)
	ListUseCases(ctx context.Context) ([]UseCase, error)
	ListWorkflows(ctx context.Context, useCase string) ([]Workflow, error)
	GetChecklist(ctx context.Context, useCase, blueprint string) ([]ChecklistTopic, error)
	ListWebhookEvents(ctx context.Context) ([]WebhookEventGroup, error)
	ListWebhooks(ctx context.Context) ([]WebhookConfig, error)
	CreateWebhook(ctx context.Context, p *WebhookParams) (map[string]any, error)
	UpdateWebhook(ctx context.Context, p *WebhookParams) (map[string]any, error)
	ListTestCases(ctx context.Context, blueprint string) ([]TestCase, error)
	RunEvaluation(ctx context.Context, evalKey string, req EvalRequest) (*EvalResponse, error)
}
