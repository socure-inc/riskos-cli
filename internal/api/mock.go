package api

import "context"

// MockClient is a test double implementing Client.
// Each method field can be set to return specific values.
type MockClient struct {
	VerifyAPIKeyFn      func(ctx context.Context) (*AuthVerifyResult, error)
	ListUseCasesFn      func(ctx context.Context) ([]UseCase, error)
	ListWorkflowsFn     func(ctx context.Context, useCase string) ([]Workflow, error)
	GetChecklistFn      func(ctx context.Context, useCase, blueprint string) ([]ChecklistTopic, error)
	ListWebhookEventsFn func(ctx context.Context) ([]WebhookEventGroup, error)
	ListWebhooksFn      func(ctx context.Context) ([]WebhookConfig, error)
	CreateWebhookFn     func(ctx context.Context, p *WebhookParams) (map[string]any, error)
	UpdateWebhookFn     func(ctx context.Context, p *WebhookParams) (map[string]any, error)
	ListTestCasesFn     func(ctx context.Context, useCase string) ([]TestCase, error)
	RunEvaluationFn     func(ctx context.Context, evalKey string, req EvalRequest) (*EvalResponse, error)
}

func (m *MockClient) VerifyAPIKey(ctx context.Context) (*AuthVerifyResult, error) {
	if m.VerifyAPIKeyFn != nil {
		return m.VerifyAPIKeyFn(ctx)
	}
	return nil, nil
}

func (m *MockClient) ListUseCases(ctx context.Context) ([]UseCase, error) {
	if m.ListUseCasesFn != nil {
		return m.ListUseCasesFn(ctx)
	}
	return nil, nil
}

func (m *MockClient) ListWorkflows(ctx context.Context, useCase string) ([]Workflow, error) {
	if m.ListWorkflowsFn != nil {
		return m.ListWorkflowsFn(ctx, useCase)
	}
	return nil, nil
}

func (m *MockClient) GetChecklist(ctx context.Context, useCase, blueprint string) ([]ChecklistTopic, error) {
	if m.GetChecklistFn != nil {
		return m.GetChecklistFn(ctx, useCase, blueprint)
	}
	return nil, nil
}

func (m *MockClient) ListWebhookEvents(ctx context.Context) ([]WebhookEventGroup, error) {
	if m.ListWebhookEventsFn != nil {
		return m.ListWebhookEventsFn(ctx)
	}
	return nil, nil
}

func (m *MockClient) ListWebhooks(ctx context.Context) ([]WebhookConfig, error) {
	if m.ListWebhooksFn != nil {
		return m.ListWebhooksFn(ctx)
	}
	return nil, nil
}

func (m *MockClient) CreateWebhook(ctx context.Context, p *WebhookParams) (map[string]any, error) {
	if m.CreateWebhookFn != nil {
		return m.CreateWebhookFn(ctx, p)
	}
	return nil, nil
}

func (m *MockClient) UpdateWebhook(ctx context.Context, p *WebhookParams) (map[string]any, error) {
	if m.UpdateWebhookFn != nil {
		return m.UpdateWebhookFn(ctx, p)
	}
	return nil, nil
}

func (m *MockClient) ListTestCases(ctx context.Context, useCase string) ([]TestCase, error) {
	if m.ListTestCasesFn != nil {
		return m.ListTestCasesFn(ctx, useCase)
	}
	return nil, nil
}

func (m *MockClient) RunEvaluation(ctx context.Context, evalKey string, req EvalRequest) (*EvalResponse, error) {
	if m.RunEvaluationFn != nil {
		return m.RunEvaluationFn(ctx, evalKey, req)
	}
	return nil, nil
}
