package api

// Client is the interface for all RiskOS API operations.
// It is satisfied by *RiskOSClient (production) and MockClient (tests).
type Client interface {
	VerifyAPIKey() (*AuthVerifyResult, error)
	ListUseCases() ([]UseCase, error)
	ListWorkflows(useCase string) ([]Workflow, error)
	GetChecklist(useCase, blueprint string) ([]ChecklistTopic, error)
	ListWebhookEvents() ([]WebhookEventGroup, error)
	ListWebhooks() ([]WebhookConfig, error)
	CreateWebhook(p *WebhookParams) (map[string]any, error)
	UpdateWebhook(p *WebhookParams) (map[string]any, error)
	ListTestCases(blueprint string) ([]TestCase, error)
	RunEvaluation(evalKey string, req EvalRequest) (*EvalResponse, error)
}
