package cmd

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/effectivai/riskos-cli/internal/api"
	"github.com/effectivai/riskos-cli/internal/debug"
	"github.com/effectivai/riskos-cli/internal/ui"
)

const (
	specURLAPI     = "https://raw.githubusercontent.com/socure-inc/.github/refs/heads/main/openapi-spec/riskos-api-spec.json"
	specURLWebhook = "https://raw.githubusercontent.com/socure-inc/.github/refs/heads/main/openapi-spec/riskos-webhooks-spec.json"
)

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

			if blueprint == "" {
				spin := ui.NewSpinner("Resolving blueprint…")
				spin.Start()
				var err error
				blueprint, err = resolveBlueprintForUseCase(cmd.Context(), client, useCase)
				spin.Stop()
				if err != nil {
					ui.Abort("Cannot resolve blueprint: "+err.Error(), "", "")
				}
			}

			spin := ui.NewSpinner("Downloading integration guide…")
			spin.Start()
			topics, err := client.GetChecklist(cmd.Context(), useCase, blueprint)
			spin.Stop()
			if err != nil {
				ui.Abort("Failed to download guide: "+err.Error(), "", "")
			}

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
					sb.WriteString(strings.TrimSpace(decodeStepContent(step)))
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
				"riskos asset openapi ./openapi.json",
				"riskos asset testcases " + useCase + " ./testcases.json",
			})
			ui.Outro("Guide ready.")
			return nil
		},
	}

	assetOpenAPICmd := &cobra.Command{
		Use:     "openapi [target]",
		Short:   "Download the RiskOS API OpenAPI specification",
		Example: "  riskos asset openapi ./openapi.json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			output := resolveOutput(cmd)

			spin := ui.NewSpinner("Downloading API spec…")
			spin.Start()
			content, err := downloadSpec(cmd, specURLAPI)
			spin.Stop()
			if err != nil {
				ui.Abort("Failed to download spec: "+err.Error(), "", "")
			}

			if err := writeFile(target, content); err != nil {
				ui.Abort("Failed to write file: "+err.Error(), "", "")
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{"status": "success", "path": target, "source": specURLAPI})
				return nil
			}

			ui.Intro("riskos asset openapi", "")
			ui.StepDone("API specification downloaded")
			ui.StepValue("Saved to", target, ui.Success)
			ui.NextSteps([]string{
				"riskos asset webhook-spec ./webhooks.json",
				"riskos webhook events",
			})
			ui.Outro("Spec ready.")
			return nil
		},
	}

	assetWebhookSpecCmd := &cobra.Command{
		Use:     "webhook-spec [target]",
		Short:   "Download the RiskOS Webhooks OpenAPI specification",
		Example: "  riskos asset webhook-spec ./webhooks.json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			output := resolveOutput(cmd)

			spin := ui.NewSpinner("Downloading webhook spec…")
			spin.Start()
			content, err := downloadSpec(cmd, specURLWebhook)
			spin.Stop()
			if err != nil {
				ui.Abort("Failed to download spec: "+err.Error(), "", "")
			}

			if err := writeFile(target, content); err != nil {
				ui.Abort("Failed to write file: "+err.Error(), "", "")
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{"status": "success", "path": target, "source": specURLWebhook})
				return nil
			}

			ui.Intro("riskos asset webhook-spec", "")
			ui.StepDone("Webhook specification downloaded")
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
		Example: "  riskos asset testcases consumer_onboarding ./testcases.json",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			useCase, target := args[0], args[1]
			client := newClient(cmd)
			output := resolveOutput(cmd)

			spin := ui.NewSpinner("Fetching test cases…")
			spin.Start()
			tcs, err := client.ListTestCases(cmd.Context(), useCase)
			spin.Stop()
			if errors.Is(err, api.ErrFeatureNotAvailable) {
				ui.Abort("Test-case download isn't available on the connected server yet.",
					"Please try again after the next RiskOS update.", "")
			}
			if err != nil {
				ui.Abort("Failed to fetch test cases: "+err.Error(), "Check your API key:  riskos auth whoami", "")
			}
			count := len(tcs)

			// Marshal an empty (non-nil) slice as [] rather than null.
			out := tcs
			if out == nil {
				out = []api.TestCase{}
			}
			b, err := json.MarshalIndent(out, "", "  ")
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
			if count == 0 {
				ui.StepDone(fmt.Sprintf("No test cases configured for %q — wrote an empty list", useCase))
			} else {
				ui.StepDone(fmt.Sprintf("%d test cases downloaded", count))
			}
			ui.StepValue("Use case", useCase, ui.Key)
			ui.StepValue("Count", fmt.Sprintf("%d", count), ui.Accent)
			ui.StepValue("Saved to", target, ui.Success)
			ui.NextSteps([]string{
				"riskos testcase list " + useCase,
			})
			ui.Outro("Test cases ready.")
			return nil
		},
	}

	assetGuideCmd.Flags().String("blueprint", "", "Blueprint name (auto-detected if omitted)")
	assetCmd.AddCommand(assetGuideCmd, assetOpenAPICmd, assetWebhookSpecCmd, assetTestCasesCmd)
	return assetCmd
}

func downloadSpec(cmd *cobra.Command, url string) (string, error) {
	debug.Log("→ GET %s", url)
	req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet, url, nil)
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

// decodeStepContent returns a checklist step's body text ready to write into the
// guide. Markdown steps arrive base64-encoded and are decoded; any other content
// type (including plain text) is returned as-is. If a markdown step fails to
// decode, the raw value is returned so a single bad step never aborts the guide.
func decodeStepContent(step api.ChecklistStep) string {
	if strings.EqualFold(step.ContentType, "markdown") {
		decoded, err := base64.StdEncoding.DecodeString(step.ContentSecondary)
		if err != nil {
			debug.Log("guide: failed to decode markdown step %s: %v", step.ID, err)
			return step.ContentSecondary
		}
		return string(decoded)
	}
	return step.ContentSecondary
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
