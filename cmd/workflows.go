package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"riskos/internal/ui"
)

func buildWorkflowsCmd() *cobra.Command {
	workflowsCmd := &cobra.Command{
		Use:   "workflows",
		Short: "Browse workflows for a use case",
	}

	workflowsListCmd := &cobra.Command{
		Use:     "list [usecase]",
		Short:   "List workflows for a given use case",
		Example: "  riskos workflows list consumer_onboarding",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			useCase := args[0]
			client := newClient(cmd)
			output := resolveOutput(cmd)

			spin := ui.NewSpinner("Fetching workflows…")
			spin.Start()
			wfs, err := client.ListWorkflows(useCase)
			spin.Stop()
			if err != nil {
				ui.Abort("Failed to fetch workflows: "+err.Error(), "", "")
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{"useCase": useCase, "workflows": wfs})
				return nil
			}

			ui.Intro("riskos workflows list", useCase)
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

	workflowsCmd.AddCommand(workflowsListCmd)
	return workflowsCmd
}
