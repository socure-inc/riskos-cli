package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func DeliverEvent(ctx context.Context, target string, payload map[string]any) (statusCode int, latencyMs int64, err error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, 0, fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return 0, 0, fmt.Errorf("building request: %w", err)
	}

	eventType, _ := payload["event_type"].(string)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-RiskOS-Simulated", "true")
	if eventType != "" {
		req.Header.Set("X-RiskOS-Event-Type", eventType)
	}

	start := time.Now()
	resp, err := httpClient.Do(req)
	latencyMs = time.Since(start).Milliseconds()
	if err != nil {
		return 0, latencyMs, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, latencyMs, nil
}
