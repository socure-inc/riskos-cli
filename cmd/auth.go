package cmd

import (
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/effectivai/riskos-cli/internal/api"
	"github.com/effectivai/riskos-cli/internal/config"
	"github.com/effectivai/riskos-cli/internal/ui"
)

func buildAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication and API keys",
	}

	authLoginCmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate interactively and store credentials in the OS keychain",
		Long: `Authenticate with your RiskOS API keys.

Prompts for your CLI/MCP key (required) and Evaluation key (optional).
Keys are verified against the RiskOS API, then stored securely in the
OS keychain. On systems without keychain support, use environment
variables instead (RISKOS_API_KEY, RISKOS_EVAL_KEY).`,
		Example: "  riskos auth login",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			output := resolveOutput(cmd)
			baseURL, _ := cmd.Root().PersistentFlags().GetString("base-url")

			if output == "plain" {
				ui.Intro("riskos auth login", "")
			}

			// Prompt for CLI/MCP key
			fmt.Print("  CLI/MCP API key: ")
			apiKeyBytes, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				ui.Abort("Failed to read API key: "+err.Error(), "", "")
			}
			fmt.Println()
			apiKey := string(apiKeyBytes)
			if apiKey == "" {
				ui.Abort("API key cannot be empty.", "", "")
			}

			// Prompt for Evaluation key (optional)
			fmt.Print("  Evaluation API key (optional, Enter to skip): ")
			evalKeyBytes, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				ui.Abort("Failed to read key: "+err.Error(), "", "")
			}
			fmt.Println()
			evalKey := string(evalKeyBytes)

			// Verify credentials
			spin := ui.NewSpinner("Verifying…")
			spin.Start()

			creds := &config.Credentials{APIKey: apiKey, EvalKey: evalKey, BaseURL: baseURL}
			d := depsFrom(cmd)
			client := d.ClientFn(cmd, creds)

			ucs, err := client.ListUseCases(ctx)
			if err != nil {
				spin.Stop()
				ui.Abort(
					"Authentication failed: "+err.Error(),
					"Check your CLI/MCP API key in the RiskOS dashboard",
					"https://riskos.sandbox.socure.com",
				)
			}

			// Get namespace from verify endpoint
			info, verifyErr := client.VerifyAPIKey(ctx)
			spin.Stop()
			if verifyErr != nil {
				ui.Abort("Key verification failed: "+verifyErr.Error(), "", "")
			}

			creds.Namespace = info.Namespace

			// Save to keyring
			if err := d.Store.Save(creds); err != nil {
				ui.Abort("Failed to save credentials: "+err.Error(), "", "")
			}

			displayURL := creds.BaseURL
			if displayURL == "" {
				displayURL = api.DefaultBaseURL
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{
					"status":     "ok",
					"namespace":  info.Namespace,
					"baseUrl":    displayURL,
					"useCases":   len(ucs),
					"evalKeySet": evalKey != "",
					"source":     "keyring",
				})
				return nil
			}

			ui.StepDone("Authenticated")
			ui.StepValue("Namespace", info.Namespace, ui.Key)
			ui.StepValue("Base URL", displayURL, ui.Accent)
			ui.StepValue("Use cases", fmt.Sprintf("%d available", len(ucs)), ui.Val)
			if evalKey != "" {
				ui.StepValue("Eval key", "✔ stored", ui.Success)
			} else {
				ui.StepValue("Eval key", "skipped (re-run login to add)", ui.Muted)
			}
			ui.StepValue("Storage", "OS keychain", ui.Muted)
			ui.NextSteps([]string{
				"riskos usecase list",
				"riskos auth status",
			})
			ui.Outro("You're all set!")
			return nil
		},
	}

	authStatusCmd := &cobra.Command{
		Use:     "status",
		Short:   "Show current authentication status and credential source",
		Example: "  riskos auth status\n  riskos auth status -o json",
		RunE: func(cmd *cobra.Command, args []string) error {
			output := resolveOutput(cmd)
			d := depsFrom(cmd)
			source := d.Store.Source()
			creds, _ := d.Store.Load()

			if creds == nil {
				if output == "json" {
					ui.PrintJSON(map[string]any{"status": "unauthenticated", "source": string(source)})
					return nil
				}
				ui.Intro("riskos auth status", "")
				ui.StepActive(ui.Err.Render("Not authenticated"))
				ui.LogInfo("Run:  riskos auth login")
				ui.LogInfo("Or set RISKOS_API_KEY environment variable")
				ui.Bar()
				fmt.Println()
				return nil
			}

			displayURL := creds.BaseURL
			if displayURL == "" {
				displayURL = api.DefaultBaseURL
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{
					"status":     "authenticated",
					"source":     string(source),
					"namespace":  creds.Namespace,
					"baseUrl":    displayURL,
					"evalKeySet": creds.EvalKey != "",
				})
				return nil
			}

			ui.Intro("riskos auth status", "")
			ui.StepDone("Authenticated")
			ui.StepValue("Source", string(source), ui.Accent)
			ui.StepValue("Namespace", creds.Namespace, ui.Key)
			ui.StepValue("Base URL", displayURL, ui.Muted)
			if creds.EvalKey != "" {
				ui.StepValue("Eval key", "✔ set", ui.Success)
			} else {
				ui.StepValue("Eval key", "not set", ui.Muted)
			}
			ui.Bar()
			fmt.Println()
			return nil
		},
	}

	authLogoutCmd := &cobra.Command{
		Use:     "logout",
		Short:   "Remove stored credentials from keychain",
		Example: "  riskos auth logout",
		RunE: func(cmd *cobra.Command, args []string) error {
			output := resolveOutput(cmd)
			d := depsFrom(cmd)
			if err := d.Store.Delete(); err != nil {
				ui.Abort("Failed to remove credentials: "+err.Error(), "", "")
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{"status": "ok", "message": "credentials removed"})
				return nil
			}
			ui.Intro("riskos auth logout", "")
			ui.StepDone("Credentials removed from keychain")
			ui.LogInfo("Environment variables (if set) will still be used.")
			ui.Outro("Logged out.")
			return nil
		},
	}

	authCmd.AddCommand(authLoginCmd, authStatusCmd, authLogoutCmd)
	return authCmd
}
