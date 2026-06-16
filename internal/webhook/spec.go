package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
	"riskos/asset"
	"riskos/internal/debug"
)

const SpecURL = "https://raw.githubusercontent.com/socure-inc/.github/refs/heads/main/openapi-spec/riskos-webhooks-spec.json"

func downloadSpec(ctx context.Context, url string) (string, error) {
	debug.Log("→ GET %s", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	debug.Log("← %d %s", resp.StatusCode, resp.Status)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	debug.Log("← %d bytes downloaded", len(b))
	return string(b), nil
}

func LoadExamples(ctx context.Context) (map[string]map[string]any, error) {
	var raw []byte

	spec, err := downloadSpec(ctx, SpecURL)
	if err == nil && spec != "" {
		debug.Log("webhook spec loaded from remote")
		raw = []byte(spec)
	} else {
		debug.Log("webhook spec download failed (%v), using embedded fallback", err)
		raw = asset.WebhookSpecJSON
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parsing webhook spec: %w", err)
	}

	components, _ := doc["components"].(map[string]any)
	if components == nil {
		return nil, fmt.Errorf("webhook spec missing components")
	}
	examples, _ := components["examples"].(map[string]any)
	if examples == nil {
		return nil, fmt.Errorf("webhook spec missing components.examples")
	}

	result := make(map[string]map[string]any)
	for _, exObj := range examples {
		ex, _ := exObj.(map[string]any)
		if ex == nil {
			continue
		}
		value, _ := ex["value"].(map[string]any)
		if value == nil {
			continue
		}
		eventType, _ := value["event_type"].(string)
		if eventType == "" {
			continue
		}
		result[eventType] = value
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no event examples found in webhook spec")
	}
	return result, nil
}

func AvailableEventTypes(examples map[string]map[string]any) []string {
	types := make([]string, 0, len(examples))
	for t := range examples {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

func FreshPayload(examples map[string]map[string]any, eventType string) (map[string]any, error) {
	tmpl, ok := examples[eventType]
	if !ok {
		return nil, fmt.Errorf("unknown event type %q; available: %v", eventType, AvailableEventTypes(examples))
	}

	b, err := json.Marshal(tmpl)
	if err != nil {
		return nil, err
	}
	var clone map[string]any
	if err := json.Unmarshal(b, &clone); err != nil {
		return nil, err
	}

	clone["event_id"] = uuid.New().String()
	clone["event_at"] = time.Now().UTC().Format(time.RFC3339Nano)

	return clone, nil
}
