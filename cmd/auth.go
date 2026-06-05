package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"riskos/internal/api"
	"riskos/internal/config"
	"riskos/internal/ui"
)

func buildAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication and API keys",
	}

	authLoginCmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with your RiskOS CLI/MCP API key",
		Long: `Authenticate using your RiskOS CLI/MCP API key.

Two separate keys are used:
  --api-key       CLI/MCP key  — authorises all discovery and management commands
  --eval-api-key  Evaluation key — authorises testcase run (can be set later)

Both keys are Bearer tokens issued from the RiskOS dashboard.`,
		Example: `  riskos auth login --api-key <cli-mcp-key>
  riskos auth login --api-key <cli-mcp-key> --eval-api-key <eval-key>
  RISKOS_API_KEY=<key> riskos usecases list   # env var, no login needed`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, _ := cmd.Flags().GetString("api-key")
			evalKey, _ := cmd.Flags().GetString("eval-api-key")
			prod, _ := cmd.Flags().GetBool("prod")

			// Env var fallbacks
			if apiKey == "" {
				apiKey = os.Getenv("RISKOS_API_KEY")
			}
			if evalKey == "" {
				evalKey = os.Getenv("RISKOS_EVAL_KEY")
			}

			// Interactive prompt if still missing
			if apiKey == "" {
				fmt.Print("  CLI/MCP API key: ")
				fmt.Scan(&apiKey)
			}

			output := resolveOutput(cmd)
			if output == "plain" {
				ui.Intro("riskos auth login", "")
			}

			spin := ui.NewSpinner("Connecting to RiskOS…")
			spin.Start()

			creds := &config.Credentials{APIKey: apiKey, Prod: prod}
			var client api.Client
			if newClientFn != nil {
				client = newClientFn()
			} else {
				client = api.NewClientFromCreds(creds)
			}

			spin.Update("Verifying CLI/MCP key…")
			info, err := client.VerifyAPIKey()
			if err != nil {
				spin.Stop()
				ui.Abort(
					"Authentication failed: "+err.Error(),
					"Check your CLI/MCP API key in the RiskOS dashboard",
					"https://riskos.sandbox.socure.com",
				)
			}

			spin.Update("Authenticated ✔")
			spin.Stop()

			creds.Namespace = info.Namespace
			if evalKey != "" {
				creds.EvalKey = evalKey
			}
			if err := defaultStoreFn().Save(creds); err != nil {
				ui.Abort("Failed to save credentials: "+err.Error(), "", "")
			}

			env := map[bool]string{true: "production", false: "sandbox"}[prod]

			if output == "json" {
				ui.PrintJSON(map[string]any{
					"status":           "ok",
					"namespace":        info.Namespace,
					"environment":      env,
					"eval_api_key_set": evalKey != "",
				})
				return nil
			}

			ui.StepDone("Authenticated")
			ui.StepValue("Namespace", info.Namespace, ui.Key)
			ui.StepValue("Environment", env, ui.Accent)
			if evalKey != "" {
				ui.StepValue("Eval API key", "✔ set", ui.Success)
			} else {
				ui.StepValue("Eval API key", "not set", ui.Muted)
				ui.LogInfo("Set later:  riskos auth set-eval-key --api-key <uuid>")
			}
			ui.NextSteps([]string{
				"riskos usecases list",
				"riskos asset guide consumer_onboarding ./guide.md",
			})
			ui.Outro("You're all set!")
			return nil
		},
	}

	authWhoamiCmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, _ := defaultStoreFn().Load()
			if creds == nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "  Not authenticated. Run: riskos auth login --api-key <key>")
				return nil
			}
			output := resolveOutput(cmd)
			env := map[bool]string{true: "production", false: "sandbox"}[creds.Prod]
			if output == "json" {
				ui.PrintJSON(map[string]any{
					"namespace":        creds.Namespace,
					"environment":      env,
					"eval_api_key_set": creds.EvalKey != "",
					"status":           "authenticated",
				})
				return nil
			}
			ui.Intro("riskos auth whoami", "")
			ui.StepValue("Namespace", creds.Namespace, ui.Key)
			ui.StepValue("Environment", env, ui.Accent)
			if creds.EvalKey != "" {
				ui.StepValue("Eval API key", "✔ set", ui.Success)
			} else {
				ui.StepValue("Eval API key", "not set", ui.Muted)
			}
			ui.Bar()
			fmt.Println()
			return nil
		},
	}

	authSetEvalKeyCmd := &cobra.Command{
		Use:     "set-eval-key",
		Short:   "Store the Evaluation API key for running test cases",
		Example: "  riskos auth set-eval-key --api-key <eval-api-key-uuid>",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, _ := cmd.Flags().GetString("api-key")
			if key == "" {
				fmt.Print("  Evaluation API key: ")
				fmt.Scan(&key)
			}
			creds := requireAuth()
			creds.EvalKey = key
			if err := defaultStoreFn().Save(creds); err != nil {
				ui.Abort("Failed to save: "+err.Error(), "", "")
			}
			output := resolveOutput(cmd)
			if output == "json" {
				ui.PrintJSON(map[string]any{"status": "ok", "eval_api_key_set": true})
				return nil
			}
			ui.Intro("riskos auth set-eval-key", "")
			ui.StepDone("Evaluation API key stored")
			ui.NextSteps([]string{
				"riskos testcase list",
				"riskos testcase run Valid_Consumer_Accept",
			})
			ui.Outro("Ready to run test cases.")
			return nil
		},
	}

	authLoginCmd.Flags().String("api-key", "", "CLI/MCP Bearer API key  [env: RISKOS_API_KEY]")
	authLoginCmd.Flags().String("eval-api-key", "", "Evaluation API key       [env: RISKOS_EVAL_KEY]")
	authLoginCmd.Flags().Bool("prod", false, "Use production endpoint")

	authSetEvalKeyCmd.Flags().String("api-key", "", "Evaluation API key (Bearer token UUID)")

	authCmd.AddCommand(authLoginCmd, authWhoamiCmd, authSetEvalKeyCmd)
	return authCmd
}
