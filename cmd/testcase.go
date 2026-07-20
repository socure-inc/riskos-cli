package cmd

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/effectivai/riskos-cli/internal/api"
	"github.com/effectivai/riskos-cli/internal/ui"
)

type persona struct {
	Description           string
	ExpectedFinalDecision string
	IndividualID          string
}

var testPersonas = map[string]persona{
	"Valid_Consumer_Accept": {"Standard consumer — passes KYC + Watchlist", "ACCEPT", "test-valid-consumer"},
	"Watchlist_Hit_Reject":  {"Consumer that triggers an OFAC watchlist hit", "REJECT", "test-watchlist-hit"},
	"DocV_Required_Accept":  {"Consumer where DocV is triggered then passes", "ACCEPT", "test-docv-required"},
	"Fraud_Signal_Reject":   {"Consumer flagged by fraud signals", "REJECT", "test-fraud-signal"},
}

var testPersonaOrder = []string{
	"Valid_Consumer_Accept", "Watchlist_Hit_Reject", "DocV_Required_Accept", "Fraud_Signal_Reject",
}

func buildTestcaseCmd() *cobra.Command {
	testcaseCmd := &cobra.Command{Use: "testcase", Short: "Discover and execute test cases"}

	testcaseListCmd := &cobra.Command{
		Use: "list [usecase]", Short: "List available sandbox test cases",
		Example: "  riskos testcase list consumer_onboarding\n  riskos testcase list consumer_onboarding -o json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			useCase := args[0]
			client := newClient(cmd)
			output := resolveOutput(cmd)

			blueprint, err := resolveTestcaseBlueprint(cmd, client, useCase)
			if err != nil {
				ui.Abort("Cannot resolve blueprint: "+err.Error(), "", "")
			}

			spin := ui.NewSpinner("Fetching test cases…")
			spin.Start()
			tcs, err := client.ListTestCases(cmd.Context(), blueprint)
			spin.Stop()
			if err != nil {
				ui.Abort("Failed to fetch test cases: "+err.Error(), "Check your API key:  riskos auth whoami", "")
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{"testCases": nonNilTestCases(tcs), "count": len(tcs)})
				return nil
			}

			if len(tcs) == 0 {
				ui.Intro("riskos testcase list", useCase)
				ui.LogInfo(fmt.Sprintf("No sandbox test cases configured for %q.", useCase))
				ui.Bar()
				fmt.Println()
				return nil
			}

			ui.Intro("riskos testcase list", fmt.Sprintf("%d available", len(tcs)))
			for _, tc := range tcs {
				ui.StepActive(ui.Key.Render(tc.TestName) + "  " + ui.DecisionBadge(tc.Decision))
				if tc.Description != "" {
					ui.LogInfo(tc.Description)
				}
			}
			ui.NextSteps([]string{
				"riskos testcase describe " + useCase + " " + tcs[0].TestName,
			})
			fmt.Println()
			return nil
		},
	}

	testcaseDescribeCmd := &cobra.Command{
		Use: "describe [usecase] [testcase]", Short: "Show details for a test case",
		Example: "  riskos testcase describe consumer_onboarding Watchlist_Hit_Reject",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			useCase, name := args[0], args[1]
			client := newClient(cmd)
			output := resolveOutput(cmd)

			blueprint, err := resolveTestcaseBlueprint(cmd, client, useCase)
			if err != nil {
				ui.Abort("Cannot resolve blueprint: "+err.Error(), "", "")
			}

			spin := ui.NewSpinner("Fetching test cases…")
			spin.Start()
			tcs, err := client.ListTestCases(cmd.Context(), blueprint)
			spin.Stop()
			if err != nil {
				ui.Abort("Failed to fetch test cases: "+err.Error(), "Check your API key:  riskos auth whoami", "")
			}

			var found *api.TestCase
			names := make([]string, 0, len(tcs))
			for i := range tcs {
				names = append(names, tcs[i].TestName)
				if tcs[i].TestName == name {
					found = &tcs[i]
				}
			}
			if found == nil {
				s := ui.SuggestTypo(name, names)
				hint := "Run  riskos testcase list " + useCase + "  to see available cases"
				if s != "" {
					hint = "Did you mean:  riskos testcase describe " + useCase + " " + s
				}
				ui.Abort("Unknown test case: "+name, hint, "")
			}

			if output == "json" {
				ui.PrintJSON(found)
				return nil
			}

			ui.Intro("riskos testcase describe", found.TestName)
			ui.StepActive(ui.Key.Render(found.TestName))
			ui.StepValue("Use case", found.UseCase, ui.Key)
			ui.StepValue("Decision", found.Decision, ui.Accept)
			if found.Solution != "" {
				ui.StepValue("Solution", found.Solution, ui.Accent)
			}
			if found.Description != "" {
				ui.StepValue("Desc", found.Description, ui.Muted)
			}
			ui.StepValue("Config type", found.ConfigType, ui.Muted)
			ui.StepValue("Active", fmt.Sprintf("%t", found.IsActive), ui.Muted)
			ui.Bar()
			fmt.Println()
			return nil
		},
	}

	testcaseRunCmd := &cobra.Command{
		Use: "run [testcase]", Short: "Run a test case against the RiskOS sandbox",
		Long:    "Posts a real evaluation to the RiskOS sandbox and returns the live eval ID and Hosted Flow URL.",
		Example: "  riskos testcase run Valid_Consumer_Accept\n  riskos testcase run Watchlist_Hit_Reject -o json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			client := newClient(cmd)
			creds := requireAuth(cmd)
			output := resolveOutput(cmd)
			p, ok := testPersonas[name]
			if !ok {
				s := ui.SuggestTypo(name, testPersonaOrder)
				hint := "Run  riskos testcase list  to see available cases"
				if s != "" {
					hint = "Did you mean:  riskos testcase run " + s
				}
				ui.Abort("Unknown test case: "+name, hint, "")
			}
			if creds.EvalKey == "" {
				ui.Abort("No Evaluation API key.", "Run:  riskos auth set-eval-key --api-key <uuid>", "")
			}
			evalReq := api.EvalRequest{
				ID:        uuid.NewString(),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Workflow:  "consumer_onboarding",
				Data: api.EvalRequestData{
					Individual: map[string]string{"id": p.IndividualID},
					Custom:     map[string]string{"redirect_uri": "https://example.com/callback"},
				},
			}
			spin := ui.NewSpinner("Posting to sandbox evaluation API…")
			spin.Start()
			resp, err := client.RunEvaluation(cmd.Context(), creds.EvalKey, evalReq)
			spin.Stop()
			if err != nil {
				ui.Abort("Evaluation failed: "+err.Error(), "Check your Eval API key:  riskos auth whoami",
					"https://help.socure.com/riskos/docs/kyc-watchlist-screening-hosted-flow-start-an-evaluation")
			}
			launched := resp.Decision == "REVIEW" && resp.EvalStatus == "evaluation_paused"
			if output == "json" {
				ui.PrintJSON(map[string]any{"testName": name, "description": p.Description,
					"expectedFinalDecision": p.ExpectedFinalDecision, "evalResponse": resp})
				return nil
			}
			ui.Intro("riskos testcase run", name)
			ui.ProgressStep(1, 3, "POST /api/evaluation  →  evaluation created", true)
			ui.StepValue("Eval ID", resp.EvalID, ui.Muted)
			ui.StepValue("Status", resp.EvalStatus, ui.Accent)
			ui.StepValue("Workflow", resp.Workflow, ui.Key)
			ui.StepValue("Environment", resp.Environment, ui.Muted)
			ui.ProgressStep(2, 3, "Hosted Flow URL ready", false)
			if resp.RedirectURI != "" {
				fmt.Printf("  %s  %s\n", ui.Spine.Render("│ "), ui.Accent.Render(resp.RedirectURI))
			}
			ui.ProgressStep(3, 3, "Awaiting user  →  webhook fires on completion", false)
			ui.LogInfo("1.  Redirect user to the Hosted Flow URL above")
			ui.LogInfo("2.  RiskOS runs KYC / Watchlist / DocV inside the hosted UI")
			ui.LogInfo(fmt.Sprintf("3.  evaluation_completed  fires  →  expected: %s", p.ExpectedFinalDecision))
			nextTC := "Watchlist_Hit_Reject"
			for _, n := range testPersonaOrder {
				if n != name {
					nextTC = n
					break
				}
			}
			ui.NextSteps([]string{
				"riskos webhook forward --target http://localhost:3000/webhooks/riskos",
				"riskos testcase run " + nextTC,
			})
			if launched {
				ui.Outro(fmt.Sprintf("Evaluation launched  ·  expected: %s", ui.DecisionBadge(p.ExpectedFinalDecision)))
			} else {
				ui.Abort("Unexpected response — expected decision=REVIEW / eval_status=evaluation_paused",
					"Check your Eval API key:  riskos auth whoami",
					"https://help.socure.com/riskos/docs/kyc-watchlist-screening-hosted-flow-start-an-evaluation")
			}
			return nil
		},
	}

	testcaseListCmd.Flags().String("blueprint", "", "Blueprint name (auto-detected if omitted)")
	testcaseDescribeCmd.Flags().String("blueprint", "", "Blueprint name (auto-detected if omitted)")
	testcaseCmd.AddCommand(testcaseListCmd, testcaseDescribeCmd, testcaseRunCmd)
	return testcaseCmd
}

// resolveTestcaseBlueprint returns the blueprint name to query test cases for,
// honoring an explicit --blueprint flag and otherwise resolving it from the use case.
func resolveTestcaseBlueprint(cmd *cobra.Command, client api.Client, useCase string) (string, error) {
	if bp, _ := cmd.Flags().GetString("blueprint"); bp != "" {
		return bp, nil
	}
	spin := ui.NewSpinner("Resolving blueprint…")
	spin.Start()
	bp, err := resolveBlueprintForUseCase(cmd.Context(), client, useCase)
	spin.Stop()
	if err != nil {
		return "", err
	}
	if bp == "" {
		return "", fmt.Errorf("no blueprint configured for use case %q", useCase)
	}
	return bp, nil
}

// nonNilTestCases returns tcs, or an empty (non-nil) slice so JSON output renders
// as [] rather than null when there are no test cases.
func nonNilTestCases(tcs []api.TestCase) []api.TestCase {
	if tcs == nil {
		return []api.TestCase{}
	}
	return tcs
}
