package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "embed"

	"github.com/spf13/cobra"
	"riskos/internal/ui"
)

//go:embed assets/openapi.yaml
var openAPIContent string

func buildAssetCmd() *cobra.Command {
	assetCmd := &cobra.Command{
		Use:   "asset",
		Short: "Download integration assets — guides, OpenAPI specs, test cases",
	}

	assetGuideCmd := &cobra.Command{
		Use:     "guide [usecase] [target]",
		Short:   "Download the integration guide for a use case",
		Example: "  riskos asset guide consumer_onboarding ./riskos/guides/consumer_onboarding.md",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			useCase, target := args[0], args[1]
			client := newClient(cmd)
			output := resolveOutput(cmd)
			blueprint, _ := cmd.Flags().GetString("blueprint")

			// Auto-resolve blueprint from use case list
			if blueprint == "" {
				spin := ui.NewSpinner("Resolving blueprint…")
				spin.Start()
				var err error
				blueprint, err = resolveBlueprintForUseCase(client, useCase)
				spin.Stop()
				if err != nil {
					ui.Abort("Cannot resolve blueprint: "+err.Error(), "", "")
				}
			}

			spin := ui.NewSpinner("Downloading integration guide…")
			spin.Start()
			topics, err := client.GetChecklist(useCase, blueprint)
			spin.Stop()
			if err != nil {
				ui.Abort("Failed to download guide: "+err.Error(), "", "")
			}

			// Render to Markdown
			var sb strings.Builder
			fmt.Fprintf(&sb, "# RiskOS Integration Guide — %s\n\n", useCase)
			if blueprint != "" {
				fmt.Fprintf(&sb, "> Blueprint: `%s`\n\n", blueprint)
			}
			totalSteps := 0
			for _, topic := range topics {
				for _, step := range topic.Steps {
					totalSteps++
					fmt.Fprintf(&sb, "\n## %s. %s\n\n", step.ID, step.Title)
					sb.WriteString(strings.TrimSpace(step.ContentSecondary))
					sb.WriteString("\n")
				}
			}

			if err := writeFile(target, sb.String()); err != nil {
				ui.Abort("Failed to write file: "+err.Error(), "", "")
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{
					"status": "success", "message": "Downloaded Integration Guide",
					"path": target, "steps": totalSteps,
				})
				return nil
			}

			ui.Intro("riskos asset guide", "")
			ui.StepDone("Integration guide downloaded")
			ui.StepValue("Use case", useCase, ui.Key)
			ui.StepValue("Steps", fmt.Sprintf("%d", totalSteps), ui.Accent)
			ui.StepValue("Saved to", target, ui.Success)
			ui.NextSteps([]string{
				"riskos asset openapi ./riskos/openapi/openapi.yaml",
				"riskos asset testcases " + useCase + " ./riskos/testcases/" + useCase + ".json",
			})
			ui.Outro("Guide ready.")
			return nil
		},
	}

	assetOpenAPICmd := &cobra.Command{
		Use:     "openapi [target]",
		Short:   "Download the OpenAPI specification",
		Example: "  riskos asset openapi ./riskos/openapi/openapi.yaml",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			output := resolveOutput(cmd)

			if err := writeFile(target, openAPIContent); err != nil {
				ui.Abort("Failed to write file: "+err.Error(), "", "")
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{"status": "success", "path": target})
				return nil
			}

			ui.Intro("riskos asset openapi", "")
			ui.StepDone("OpenAPI specification downloaded")
			ui.StepValue("Version", "3.1.0", ui.Accent)
			ui.StepValue("Saved to", target, ui.Success)
			ui.NextSteps([]string{
				"riskos webhook events",
				"riskos webhook create --name my-hook --url https://... ...",
			})
			ui.Outro("Spec ready.")
			return nil
		},
	}

	assetTestCasesCmd := &cobra.Command{
		Use:     "testcases [usecase] [target]",
		Short:   "Download test cases for a use case",
		Example: "  riskos asset testcases consumer_onboarding ./riskos/testcases/consumer_onboarding.json",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			useCase, target := args[0], args[1]
			client := newClient(cmd)
			output := resolveOutput(cmd)
			blueprint, _ := cmd.Flags().GetString("blueprint")

			// Auto-resolve blueprint
			if blueprint == "" {
				spin := ui.NewSpinner("Resolving blueprint…")
				spin.Start()
				var err error
				blueprint, err = resolveBlueprintForUseCase(client, useCase)
				spin.Stop()
				if err != nil {
					ui.Abort("Cannot resolve blueprint: "+err.Error(), "", "")
				}
			}

			// Try fetching live test cases
			var tcs any
			count := 0
			if blueprint != "" {
				spin := ui.NewSpinner("Fetching test cases…")
				spin.Start()
				liveTCs, err := client.ListTestCases(blueprint)
				spin.Stop()
				if err == nil && len(liveTCs) > 0 {
					tcs = liveTCs
					count = len(liveTCs)
				}
			}

			// Fall back to built-in personas if none returned
			if count == 0 {
				type tc struct {
					TestName              string `json:"testName"`
					UseCase               string `json:"useCase"`
					ExpectedFinalDecision string `json:"expectedFinalDecision"`
					Description           string `json:"description"`
				}
				builtIn := []tc{
					{"Valid_Consumer_Accept", useCase, "ACCEPT", "Standard consumer — passes KYC + Watchlist"},
					{"Watchlist_Hit_Reject", useCase, "REJECT", "Consumer that triggers an OFAC watchlist hit"},
					{"DocV_Required_Accept", useCase, "ACCEPT", "Consumer where DocV is triggered then passes"},
					{"Fraud_Signal_Reject", useCase, "REJECT", "Consumer flagged by fraud signals"},
				}
				tcs = builtIn
				count = len(builtIn)
			}

			b, err := json.MarshalIndent(tcs, "", "  ")
			if err != nil {
				ui.Abort("Failed to marshal test cases: "+err.Error(), "", "")
			}
			if err := writeFile(target, string(b)); err != nil {
				ui.Abort("Failed to write file: "+err.Error(), "", "")
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{
					"status": "success", "count": count, "path": target,
				})
				return nil
			}

			ui.Intro("riskos asset testcases", "")
			ui.StepDone(fmt.Sprintf("%d test cases downloaded", count))
			ui.StepValue("Use case", useCase, ui.Key)
			ui.StepValue("Count", fmt.Sprintf("%d", count), ui.Accent)
			ui.StepValue("Saved to", target, ui.Success)
			ui.NextSteps([]string{
				"riskos testcase list",
				"riskos testcase run Valid_Consumer_Accept",
			})
			ui.Outro("Test cases ready.")
			return nil
		},
	}

	assetGuideCmd.Flags().String("blueprint", "", "Blueprint name (auto-detected if omitted)")
	assetTestCasesCmd.Flags().String("blueprint", "", "Blueprint name (auto-detected if omitted)")
	assetCmd.AddCommand(assetGuideCmd, assetOpenAPICmd, assetTestCasesCmd)
	return assetCmd
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
