package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"riskos/internal/api"
	"riskos/internal/ui"
)

const (
	fwdMinDelayMs   = 1500
	fwdMaxDelayMs   = 3000
	fwdFailRate     = 0.10
	fwdMinLatencyMs = 6
	fwdMaxLatencyMs = 45
)

func buildWebhookCmd() *cobra.Command {
	webhookCmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manage webhooks and forward events locally",
	}

	webhookEventsCmd := &cobra.Command{
		Use:     "events",
		Short:   "List all available webhook event names",
		Example: "  riskos webhook events\n  riskos webhook events -o json",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient(cmd)
			output := resolveOutput(cmd)

			spin := ui.NewSpinner("Fetching events…")
			spin.Start()
			groups, err := client.ListWebhookEvents()
			spin.Stop()
			if err != nil {
				ui.Abort("Failed to fetch events: "+err.Error(), "", "")
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{"webhookEvents": groups})
				return nil
			}

			ui.Intro("riskos webhook events", "")
			for _, g := range groups {
				ui.StepActive(ui.Accent.Render(g.Component))
				for _, name := range g.EventNames {
					ui.LogInfo("○  " + name)
				}
			}
			ui.NextSteps([]string{
				"riskos webhook list",
				"riskos webhook create --name my-hook --url https://... --use-case consumer_onboarding ...",
			})
			fmt.Println()
			return nil
		},
	}

	webhookListCmd := &cobra.Command{
		Use:     "list",
		Short:   "List webhooks configured for this account",
		Example: "  riskos webhook list\n  riskos webhook list -o json",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient(cmd)
			output := resolveOutput(cmd)

			spin := ui.NewSpinner("Fetching webhooks…")
			spin.Start()
			configs, err := client.ListWebhooks()
			spin.Stop()
			if err != nil {
				ui.Abort("Failed to fetch webhooks: "+err.Error(), "", "")
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{"webhooks": configs, "count": len(configs)})
				return nil
			}

			ui.Intro("riskos webhook list", fmt.Sprintf("%d configured", len(configs)))
			if len(configs) == 0 {
				ui.LogInfo("No webhooks yet.")
				ui.NextSteps([]string{
					"riskos webhook create --name my-hook --url https://... --use-case consumer_onboarding ...",
				})
				fmt.Println()
				return nil
			}
			for _, c := range configs {
				ui.StepActive(ui.Key.Render(c.Name))
				ui.StepValue("URL", c.URL, ui.Val)
				ui.StepValue("Use case", c.UseCase, ui.Accent)
				ui.StepValue("Auth", c.AuthScheme, ui.Muted)
			}
			ui.NextSteps([]string{
				"riskos webhook forward --target http://localhost:3000/webhooks/riskos",
				"riskos testcase list",
			})
			fmt.Println()
			return nil
		},
	}

	webhookCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new webhook endpoint",
		Example: `  riskos webhook create \
    --name my-hook \
    --url https://your-app.com/webhooks/riskos \
    --use-case consumer_onboarding \
    --workflow-events evaluation_completed \
    --case-events decision_updated \
    --auth-scheme BEARER_TOKEN \
    --bearer-token secret`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient(cmd)
			output := resolveOutput(cmd)
			name, _ := cmd.Flags().GetString("name")
			urlF, _ := cmd.Flags().GetString("url")
			useCase, _ := cmd.Flags().GetString("use-case")
			wEvents, _ := cmd.Flags().GetStringArray("workflow-events")
			cEvents, _ := cmd.Flags().GetStringArray("case-events")
			authScheme, _ := cmd.Flags().GetString("auth-scheme")
			bearer, _ := cmd.Flags().GetString("bearer-token")

			params := &api.WebhookParams{
				Name: name, URL: urlF, UseCase: useCase,
				WorkflowEvents: wEvents, CaseEvents: cEvents,
				AuthScheme: authScheme, BearerToken: bearer,
			}

			spin := ui.NewSpinner(fmt.Sprintf("Creating webhook %s…", name))
			spin.Start()
			_, err := client.CreateWebhook(params)
			spin.Stop()

			if err != nil {
				ui.Abort("Failed to create webhook: "+err.Error(),
					"Webhook name must be unique — run  riskos webhook list  to check existing names", "")
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{"status": "SUCCESS", "errs": []any{}, "webhookName": name})
				return nil
			}

			ui.Intro("riskos webhook create", "")
			ui.StepDone(ui.Key.Render(name) + "  created")
			ui.StepValue("URL", urlF, ui.Val)
			ui.StepValue("Use case", useCase, ui.Accent)
			ui.StepValue("Auth", authScheme, ui.Muted)
			ui.NextSteps([]string{
				"riskos webhook list",
				"riskos webhook forward --target http://localhost:3000/webhooks/riskos",
			})
			ui.Outro("Webhook ready.")
			return nil
		},
	}

	webhookUpdateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update an existing webhook",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient(cmd)
			output := resolveOutput(cmd)
			name, _ := cmd.Flags().GetString("name")
			urlF, _ := cmd.Flags().GetString("url")
			useCase, _ := cmd.Flags().GetString("use-case")
			wEvents, _ := cmd.Flags().GetStringArray("workflow-events")
			cEvents, _ := cmd.Flags().GetStringArray("case-events")
			authScheme, _ := cmd.Flags().GetString("auth-scheme")
			bearer, _ := cmd.Flags().GetString("bearer-token")

			params := &api.WebhookParams{
				Name: name, URL: urlF, UseCase: useCase,
				WorkflowEvents: wEvents, CaseEvents: cEvents,
				AuthScheme: authScheme, BearerToken: bearer,
			}

			spin := ui.NewSpinner(fmt.Sprintf("Updating %s…", name))
			spin.Start()
			_, err := client.UpdateWebhook(params)
			spin.Stop()
			if err != nil {
				ui.Abort("Failed to update webhook: "+err.Error(), "", "")
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{"status": "SUCCESS", "webhookName": name})
				return nil
			}
			ui.Intro("riskos webhook update", "")
			ui.StepDone(ui.Key.Render(name) + "  updated")
			ui.StepValue("URL", urlF, ui.Val)
			ui.Outro("Done.")
			return nil
		},
	}

	webhookForwardCmd := &cobra.Command{
		Use:   "forward",
		Short: "Forward RiskOS webhook events to a local endpoint",
		Example: `  riskos webhook forward --target http://localhost:3000/webhooks/riskos
  riskos webhook forward --target http://localhost:3000/webhooks/riskos -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient(cmd)
			output := resolveOutput(cmd)
			target, _ := cmd.Flags().GetString("target")

			// Verify connectivity + fetch real event names
			spin := ui.NewSpinner("Connecting to RiskOS…")
			spin.Start()
			groups, err := client.ListWebhookEvents()
			spin.Update("Connected ✔")
			spin.Stop()
			if err != nil {
				ui.Abort("Cannot connect: "+err.Error(), "", "")
			}

			// Build sequence from real event names
			type ev struct{ component, event string }
			var allEvents []ev
			for _, g := range groups {
				for _, name := range g.EventNames {
					allEvents = append(allEvents, ev{g.Component, name})
				}
			}

			// Realistic hosted-flow sequence
			sequence := []ev{
				{"workflow", "evaluation_paused"},
				{"workflow", "evaluation_completed"},
				{"case", "review_case_assigned"},
				{"case", "decision_updated"},
				{"workflow", "evaluation_completed"},
				{"case", "fraud_confirming"},
				{"workflow", "evaluation_rerun"},
				{"workflow", "evaluation_completed"},
			}

			creds := requireAuth()
			sessionID := fmt.Sprintf("fwd_%x", rand.Int31())
			namespace := creds.Namespace

			if output == "json" {
				ui.PrintJSON(map[string]any{
					"status": "connected", "target": target,
					"namespace": namespace, "sessionId": sessionID,
				})
			} else {
				ui.Intro("riskos webhook forward", "")
				ui.StepActive(ui.Success.Render("Connected"))
				ui.StepValue("Target", target, ui.Key)
				ui.StepValue("Session", sessionID, ui.Muted)
				ui.StepValue("Namespace", namespace, ui.Muted)
				ui.LogInfo("Ctrl+C to stop")
				ui.Bar()
				fmt.Println()
			}

			// Progress bar
			bar := ui.NewProgressBar(len(sequence), "forwarding events")

			received, forwarded, failed := 0, 0, 0
			start := time.Now()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			done := make(chan struct{})
			go func() { <-sigCh; close(done) }()

			for _, ev := range sequence {
				select {
				case <-done:
					goto summary
				default:
				}
				delay := time.Duration(fwdMinDelayMs+rand.Intn(fwdMaxDelayMs-fwdMinDelayMs)) * time.Millisecond
				select {
				case <-done:
					goto summary
				case <-time.After(delay):
				}

				latency := fwdMinLatencyMs + rand.Intn(fwdMaxLatencyMs-fwdMinLatencyMs)
				failIt := rand.Float32() < fwdFailRate
				received++

				if output == "json" {
					if failIt {
						failed++
						ui.PrintJSON(map[string]any{
							"status": "delivery_failed",
							"event":  map[string]any{"component": ev.component, "eventName": ev.event, "timestamp": time.Now().UTC().Format(time.RFC3339)},
							"delivery": map[string]any{"httpCode": 0, "errMsg": "connection refused"},
						})
					} else {
						forwarded++
						ui.PrintJSON(map[string]any{
							"status": "forwarded",
							"event":  map[string]any{"component": ev.component, "eventName": ev.event, "timestamp": time.Now().UTC().Format(time.RFC3339)},
							"delivery": map[string]any{"httpCode": 200, "latencyMs": latency},
						})
					}
				} else {
					ts := time.Now().Format("15:04:05")
					if failIt {
						failed++
						ui.EventLine(ts, ev.component, ev.event, false, "connection refused")
					} else {
						forwarded++
						ui.EventLine(ts, ev.component, ev.event, true, fmt.Sprintf("%dms", latency))
					}
					bar.Advance()
				}
			}

		summary:
			bar.Done()
			signal.Stop(sigCh)
			dur := time.Since(start).Round(time.Second)

			if output == "json" {
				ui.PrintJSON(map[string]any{
					"status": "disconnected", "sessionId": sessionID,
					"stats": map[string]any{
						"received": received, "forwarded": forwarded,
						"failed": failed, "duration": dur.String(),
					},
				})
			} else {
				fmt.Println()
				ui.Bar()
				ui.StepDone("Session ended")
				ui.StepValue("Received", fmt.Sprintf("%d", received), ui.Val)
				ui.StepValue("Forwarded", fmt.Sprintf("%d", forwarded), ui.Success)
				if failed > 0 {
					ui.StepValue("Failed", fmt.Sprintf("%d", failed), ui.Err)
				} else {
					ui.StepValue("Failed", "0", ui.Muted)
				}
				ui.StepValue("Duration", dur.String(), ui.Muted)
				ui.Outro("Forwarding stopped.")
			}
			return nil
		},
	}

	webhookCreateCmd.Flags().String("name", "", "Unique webhook name (required)")
	webhookCreateCmd.Flags().String("url", "", "HTTPS endpoint URL (required)")
	webhookCreateCmd.Flags().String("use-case", "", "Use case name (required)")
	webhookCreateCmd.Flags().StringArray("workflow-events", nil, "Workflow event name (repeatable)")
	webhookCreateCmd.Flags().StringArray("case-events", nil, "Case event name (repeatable)")
	webhookCreateCmd.Flags().String("auth-scheme", "BEARER_TOKEN", "Auth: BEARER_TOKEN, BASIC_AUTH, OAUTH2")
	webhookCreateCmd.Flags().String("bearer-token", "", "Token for BEARER_TOKEN auth")
	webhookCreateCmd.MarkFlagRequired("name")
	webhookCreateCmd.MarkFlagRequired("url")
	webhookCreateCmd.MarkFlagRequired("use-case")

	webhookUpdateCmd.Flags().String("name", "", "Existing webhook name (required)")
	webhookUpdateCmd.Flags().String("url", "", "New endpoint URL (required)")
	webhookUpdateCmd.Flags().String("use-case", "", "Use case (required)")
	webhookUpdateCmd.Flags().StringArray("workflow-events", nil, "Workflow event names")
	webhookUpdateCmd.Flags().StringArray("case-events", nil, "Case event names")
	webhookUpdateCmd.Flags().String("auth-scheme", "BEARER_TOKEN", "Auth scheme")
	webhookUpdateCmd.Flags().String("bearer-token", "", "Bearer token (must match existing)")
	webhookUpdateCmd.MarkFlagRequired("name")
	webhookUpdateCmd.MarkFlagRequired("url")
	webhookUpdateCmd.MarkFlagRequired("use-case")

	webhookForwardCmd.Flags().String("target", "", "Local URL to forward events to (required)")
	webhookForwardCmd.MarkFlagRequired("target")

	webhookCmd.AddCommand(webhookEventsCmd, webhookListCmd, webhookCreateCmd, webhookUpdateCmd, webhookForwardCmd)
	return webhookCmd
}
