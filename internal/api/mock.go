package api

// MockClient is a test double implementing Client.
// Each method field can be set to return specific values.
type MockClient struct {
	VerifyAPIKeyFn      func() (*AuthVerifyResult, error)
	ListUseCasesFn      func() ([]UseCase, error)
	ListWorkflowsFn     func(string) ([]Workflow, error)
	GetChecklistFn      func(string, string) ([]ChecklistTopic, error)
	ListWebhookEventsFn func() ([]WebhookEventGroup, error)
	ListWebhooksFn      func() ([]WebhookConfig, error)
	CreateWebhookFn     func(*WebhookParams) (map[string]any, error)
	UpdateWebhookFn     func(*WebhookParams) (map[string]any, error)
	ListTestCasesFn     func(string) ([]TestCase, error)
	RunEvaluationFn     func(string, EvalRequest) (*EvalResponse, error)
}

func (m *MockClient) VerifyAPIKey() (*AuthVerifyResult, error) {
	if m.VerifyAPIKeyFn != nil {
		return m.VerifyAPIKeyFn()
	}
	return nil, nil
}

func (m *MockClient) ListUseCases() ([]UseCase, error) {
	if m.ListUseCasesFn != nil {
		return m.ListUseCasesFn()
	}
	return nil, nil
}

func (m *MockClient) ListWorkflows(useCase string) ([]Workflow, error) {
	if m.ListWorkflowsFn != nil {
		return m.ListWorkflowsFn(useCase)
	}
	return nil, nil
}

func (m *MockClient) GetChecklist(useCase, blueprint string) ([]ChecklistTopic, error) {
	if m.GetChecklistFn != nil {
		return m.GetChecklistFn(useCase, blueprint)
	}
	return nil, nil
}

func (m *MockClient) ListWebhookEvents() ([]WebhookEventGroup, error) {
	if m.ListWebhookEventsFn != nil {
		return m.ListWebhookEventsFn()
	}
	return nil, nil
}

func (m *MockClient) ListWebhooks() ([]WebhookConfig, error) {
	if m.ListWebhooksFn != nil {
		return m.ListWebhooksFn()
	}
	return nil, nil
}

func (m *MockClient) CreateWebhook(p *WebhookParams) (map[string]any, error) {
	if m.CreateWebhookFn != nil {
		return m.CreateWebhookFn(p)
	}
	return nil, nil
}

func (m *MockClient) UpdateWebhook(p *WebhookParams) (map[string]any, error) {
	if m.UpdateWebhookFn != nil {
		return m.UpdateWebhookFn(p)
	}
	return nil, nil
}

func (m *MockClient) ListTestCases(blueprint string) ([]TestCase, error) {
	if m.ListTestCasesFn != nil {
		return m.ListTestCasesFn(blueprint)
	}
	return nil, nil
}

func (m *MockClient) RunEvaluation(evalKey string, req EvalRequest) (*EvalResponse, error) {
	if m.RunEvaluationFn != nil {
		return m.RunEvaluationFn(evalKey, req)
	}
	return nil, nil
}
