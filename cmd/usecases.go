package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"riskos/internal/ui"
)

func buildUsecasesCmd() *cobra.Command {
	usecasesCmd := &cobra.Command{
		Use:   "usecases",
		Short: "Browse available use cases",
	}

	usecasesListCmd := &cobra.Command{
		Use:     "list",
		Short:   "List use cases configured for this account",
		Example: "  riskos usecases list\n  riskos usecases list -o json",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient(cmd)
			output := resolveOutput(cmd)

			spin := ui.NewSpinner("Fetching use cases…")
			spin.Start()
			ucs, err := client.ListUseCases()
			spin.Stop()
			if err != nil {
				ui.Abort("Failed to fetch use cases: "+err.Error(), "Check your API key:  riskos auth whoami", "")
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{"useCases": ucs})
				return nil
			}

			ui.Intro("riskos usecases list", fmt.Sprintf("%d found", len(ucs)))
			for _, uc := range ucs {
				ui.StepActive(ui.Key.Render(uc.Name) + "  " + ui.Muted.Render(uc.DisplayName))
				ui.StepValue("Blueprint", uc.BlueprintName, ui.Muted)
			}

			nextUC := "consumer_onboarding"
			if len(ucs) > 0 {
				nextUC = ucs[0].Name
			}
			ui.NextSteps([]string{
				"riskos workflows list " + nextUC,
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

	usecasesCmd.AddCommand(usecasesListCmd)
	return usecasesCmd
}
