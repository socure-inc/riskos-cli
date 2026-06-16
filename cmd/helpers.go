package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/effectivai/riskos-cli/internal/api"
)

// resolveBlueprintForUseCase looks up the blueprint name for a given use case
// by fetching the full use-cases list from the API.
func resolveBlueprintForUseCase(ctx context.Context, client api.Client, useCase string) (string, error) {
	ucs, err := client.ListUseCases(ctx)
	if err != nil {
		return "", fmt.Errorf("listing use cases: %w", err)
	}
	var names []string
	for _, uc := range ucs {
		if uc.Name == useCase {
			return uc.BlueprintName, nil
		}
		names = append(names, uc.Name)
	}
	if len(names) == 0 {
		return "", fmt.Errorf("use case %q not found (no use cases configured)", useCase)
	}
	return "", fmt.Errorf("use case %q not found; available: %s", useCase, strings.Join(names, ", "))
}
