package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"riskos/internal/api"
	"riskos/internal/ui"
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
		Use: "list", Short: "List available sandbox test cases",
		Example: "  riskos testcase list\n  riskos testcase list -o json",
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()
			output := resolveOutput(cmd)
			type tcRow struct {
				TestName              string `json:"testName"`
				UseCase               string `json:"useCase"`
				ExpectedFinalDecision string `json:"expectedFinalDecision"`
				Description           string `json:"description"`
			}
			var rows []tcRow
			for _, name := range testPersonaOrder {
				p := testPersonas[name]
				rows = append(rows, tcRow{name, "consumer_onboarding", p.ExpectedFinalDecision, p.Description})
			}
			if output == "json" {
				ui.PrintJSON(map[string]any{"testCases": rows, "count": len(rows)})
				return nil
			}
			ui.Intro("riskos testcase list", fmt.Sprintf("%d available", len(rows)))
			for _, r := range rows {
				ui.StepActive(ui.Key.Render(r.TestName) + "  " + ui.DecisionBadge(r.ExpectedFinalDecision))
				ui.LogInfo(r.Description)
			}
			ui.NextSteps([]string{"riskos testcase run Valid_Consumer_Accept", "riskos testcase describe Watchlist_Hit_Reject"})
			fmt.Println()
			return nil
		},
	}

	testcaseDescribeCmd := &cobra.Command{
		Use: "describe [testcase]", Short: "Show details for a test case",
		Example: "  riskos testcase describe Watchlist_Hit_Reject",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			requireAuth()
			p, ok := testPersonas[name]
			if !ok {
				s := ui.SuggestTypo(name, testPersonaOrder)
				hint := "Run  riskos testcase list  to see available cases"
				if s != "" {
					hint = "Did you mean:  riskos testcase describe " + s
				}
				ui.Abort("Unknown test case: "+name, hint, "")
			}
			ui.Intro("riskos testcase describe", name)
			ui.StepActive(ui.Key.Render(name))
			ui.StepValue("Use case", "consumer_onboarding", ui.Key)
			ui.StepValue("Expected", p.ExpectedFinalDecision, ui.Accept)
			ui.StepValue("Desc", p.Description, ui.Muted)
			ui.Note("Request payload", []string{
				`{`, `  "id": "<uuid>",`, `  "timestamp": "<iso8601>",`,
				`  "workflow": "consumer_onboarding",`, `  "data": {`,
				`    "individual": { "id": "` + p.IndividualID + `" },`,
				`    "custom": { "redirect_uri": "https://example.com/callback" }`,
				`  }`, `}`,
			})
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
			creds := requireAuth()
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
				ID:        fmt.Sprintf("%x-%x", time.Now().UnixNano(), time.Now().UnixNano()>>16),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Workflow:  "consumer_onboarding",
				Data: api.EvalRequestData{
					Individual: map[string]string{"id": p.IndividualID},
					Custom:     map[string]string{"redirect_uri": "https://example.com/callback"},
				},
			}
			spin := ui.NewSpinner("Posting to sandbox evaluation API…")
			spin.Start()
			resp, err := client.RunEvaluation(creds.EvalKey, evalReq)
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

	testcaseCmd.AddCommand(testcaseListCmd, testcaseDescribeCmd, testcaseRunCmd)
	return testcaseCmd
}
