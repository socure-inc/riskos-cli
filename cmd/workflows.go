package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"riskos/internal/ui"
)

func buildWorkflowCmd() *cobra.Command {
	workflowCmd := &cobra.Command{
		Use:   "workflow",
		Short: "Browse workflows for a use case",
	}

	workflowListCmd := &cobra.Command{
		Use:     "list [usecase]",
		Short:   "List workflows for a given use case",
		Example: "  riskos workflow list consumer_onboarding",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			useCase := args[0]
			client := newClient(cmd)
			output := resolveOutput(cmd)

			spin := ui.NewSpinner("Fetching workflows…")
			spin.Start()
			wfs, err := client.ListWorkflows(ctx, useCase)
			spin.Stop()
			if err != nil {
				ui.Abort("Failed to fetch workflows: "+err.Error(), "", "")
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{"useCase": useCase, "workflows": wfs})
				return nil
			}

			ui.Intro("riskos workflow list", useCase)
			for _, w := range wfs {
				ui.StepActive(ui.Key.Render(w.DisplayName))
				ui.StepValue("ID", w.ID, ui.Muted)
				ui.StepValue("Version", w.Version, ui.Accent)
				ui.StepValue("Namespace", w.Namespace, ui.Muted)
			}
			ui.NextSteps([]string{
				"riskos asset guide " + useCase + " ./riskos/guides/" + useCase + ".md",
				"riskos asset openapi ./riskos/openapi/openapi.yaml",
			})
			noun := "workflows"
			if len(wfs) == 1 {
				noun = "workflow"
			}
			ui.Outro(fmt.Sprintf("%d %s", len(wfs), noun))
			return nil
		},
	}

	workflowCmd.AddCommand(workflowListCmd)
	return workflowCmd
}
