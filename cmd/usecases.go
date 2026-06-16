package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"riskos/internal/ui"
)

func buildUsecaseCmd() *cobra.Command {
	usecaseCmd := &cobra.Command{
		Use:   "usecase",
		Short: "Browse available use cases",
	}

	usecaseListCmd := &cobra.Command{
		Use:     "list",
		Short:   "List use cases configured for this account",
		Example: "  riskos usecase list\n  riskos usecase list -o json",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client := newClient(cmd)
			output := resolveOutput(cmd)

			spin := ui.NewSpinner("Fetching use cases…")
			spin.Start()
			ucs, err := client.ListUseCases(ctx)
			spin.Stop()
			if err != nil {
				ui.Abort("Failed to fetch use cases: "+err.Error(), "Check your API key:  riskos auth whoami", "")
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{"useCases": ucs})
				return nil
			}

			ui.Intro("riskos usecase list", fmt.Sprintf("%d found", len(ucs)))
			for _, uc := range ucs {
				ui.StepActive(ui.Key.Render(uc.Name) + "  " + ui.Muted.Render(uc.DisplayName))
				ui.StepValue("Blueprint", uc.BlueprintName, ui.Muted)
			}

			nextUC := "consumer_onboarding"
			if len(ucs) > 0 {
				nextUC = ucs[0].Name
			}
			ui.NextSteps([]string{
				"riskos workflow list " + nextUC,
				"riskos asset guide " + nextUC + " ./guide.md",
			})
			noun := "use cases"
			if len(ucs) == 1 {
				noun = "use case"
			}
			ui.Outro(fmt.Sprintf("%d %s", len(ucs), noun))
			return nil
		},
	}

	usecaseCmd.AddCommand(usecaseListCmd)
	return usecaseCmd
}
