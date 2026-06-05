package cmd

import (
	"fmt"

	"riskos/internal/api"
)

// resolveBlueprintForUseCase looks up the blueprint name for a given use case
// by fetching the full use-cases list from the API.
func resolveBlueprintForUseCase(client api.Client, useCase string) (string, error) {
	ucs, err := client.ListUseCases()
	if err != nil {
		return "", fmt.Errorf("listing use cases: %w", err)
	}
	for _, uc := range ucs {
		if uc.Name == useCase {
			return uc.BlueprintName, nil
		}
	}
	return "", nil
}
