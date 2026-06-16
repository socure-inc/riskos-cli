package cmd

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"riskos/internal/api"
	"riskos/internal/ui"
	"riskos/internal/webhook"
)

const (
	fwdMinDelayMs = 1500
	fwdMaxDelayMs = 3000
)

// resolveBearerToken reads the bearer token from flag, file, or stdin prompt.
func resolveBearerToken(cmd *cobra.Command) string {
	token, _ := cmd.Flags().GetString("bearer-token")
	if token != "" {
		return token
	}
	tokenFile, _ := cmd.Flags().GetString("bearer-token-file")
	if tokenFile != "" {
		data, err := os.ReadFile(tokenFile)
		if err != nil {
			ui.Abort("Cannot read bearer token file: "+err.Error(), "", "")
		}
		return strings.TrimSpace(string(data))
	}
	// If auth scheme is BEARER_TOKEN, prompt interactively
	scheme, _ := cmd.Flags().GetString("auth-scheme")
	if scheme == "BEARER_TOKEN" {
		if term.IsTerminal(int(syscall.Stdin)) {
			fmt.Print("  Bearer token: ")
			b, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				ui.Abort("Failed to read token: "+err.Error(), "", "")
			}
			fmt.Println()
			return string(b)
		}
		// Non-interactive: read a single line from stdin
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			return strings.TrimSpace(scanner.Text())
		}
	}
	return ""
}

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
			groups, err := client.ListWebhookEvents(cmd.Context())
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
			configs, err := client.ListWebhooks(cmd.Context())
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
			bearer := resolveBearerToken(cmd)

			params := &api.WebhookParams{
				Name: name, URL: urlF, UseCase: useCase,
				WorkflowEvents: wEvents, CaseEvents: cEvents,
				AuthScheme: authScheme, BearerToken: bearer,
			}

			spin := ui.NewSpinner(fmt.Sprintf("Creating webhook %s…", name))
			spin.Start()
			_, err := client.CreateWebhook(cmd.Context(), params)
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
				"riskos webhook simulate --target http://localhost:3000/webhooks/riskos",
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
			bearer := resolveBearerToken(cmd)

			params := &api.WebhookParams{
				Name: name, URL: urlF, UseCase: useCase,
				WorkflowEvents: wEvents, CaseEvents: cEvents,
				AuthScheme: authScheme, BearerToken: bearer,
			}

			spin := ui.NewSpinner(fmt.Sprintf("Updating %s…", name))
			spin.Start()
			_, err := client.UpdateWebhook(cmd.Context(), params)
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

	webhookSimulateCmd := &cobra.Command{
		Use:   "simulate",
		Short: "Simulate webhook event deliveries to a local endpoint",
		Long: `Deliver simulated RiskOS webhook events to a local endpoint for development.

Event payloads are generated from the official RiskOS OpenAPI webhook specification.
All values are synthetic examples intended to demonstrate the STRUCTURE of each event
type — they do not represent real production data or evaluations.

Use this command to verify that your webhook handler correctly parses each event type,
responds with appropriate HTTP status codes, and handles the expected payload shape.`,
		Example: `  # Deliver 8 simulated events to a local server
  riskos webhook simulate --target http://localhost:3000/webhooks/riskos

  # Send a single specific event type
  riskos webhook simulate --target http://localhost:3000/webhooks/riskos --event evaluation_completed

  # Print payloads without making HTTP requests
  riskos webhook simulate --dry-run --event decision_update

  # Simulate 20 events in JSON output mode
  riskos webhook simulate --target http://localhost:3000/webhooks/riskos --events 20 -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			output := resolveOutput(cmd)
			target, _ := cmd.Flags().GetString("target")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			singleEvent, _ := cmd.Flags().GetString("event")

			if !dryRun && target == "" {
				ui.Abort("--target is required (or use --dry-run to print without delivering)", "", "")
			}

			spin := ui.NewSpinner("Loading webhook specification…")
			spin.Start()
			examples, err := webhook.LoadExamples(cmd.Context())
			spin.Stop()
			if err != nil {
				ui.Abort("Cannot load webhook spec: "+err.Error(), "", "")
			}

			allTypes := webhook.AvailableEventTypes(examples)
			sessionID := fmt.Sprintf("sim_%x", rand.Int31())

			// Single-event mode
			if singleEvent != "" {
				payload, err := webhook.FreshPayload(examples, singleEvent)
				if err != nil {
					ui.Abort(err.Error(),
						fmt.Sprintf("Available event types: %s", strings.Join(allTypes, ", ")), "")
				}

				if dryRun {
					if output == "json" {
						ui.PrintJSON(map[string]any{
							"status":    "dry_run",
							"simulated": true,
							"payload":   payload,
						})
					} else {
						ui.Intro("riskos webhook simulate", "dry-run")
						ui.StepActive(ui.Warn.Render("Simulated payload") + " — example data from OpenAPI spec, not real values")
						ui.Bar()
						fmt.Println()
						ui.PrintJSON(payload)
						fmt.Println()
						ui.Outro("Payload printed (not delivered).")
					}
					return nil
				}

				if output != "json" {
					ui.Intro("riskos webhook simulate", "single event")
					ui.StepActive(ui.Warn.Render("Simulated") + " — example payload from OpenAPI spec, not real data")
					ui.StepValue("Target", target, ui.Key)
					ui.StepValue("Event", singleEvent, ui.Accent)
					ui.Bar()
					fmt.Println()
				}

				code, latency, deliverErr := webhook.DeliverEvent(cmd.Context(), target, payload)
				if output == "json" {
					result := map[string]any{
						"status":    "delivered",
						"simulated": true,
						"event":     singleEvent,
						"payload":   payload,
						"delivery":  map[string]any{"httpCode": code, "latencyMs": latency},
					}
					if deliverErr != nil {
						result["status"] = "delivery_failed"
						result["delivery"].(map[string]any)["errMsg"] = deliverErr.Error()
					}
					ui.PrintJSON(result)
				} else {
					ts := time.Now().Format("15:04:05")
					if deliverErr != nil {
						ui.EventLine(ts, "", singleEvent, false, deliverErr.Error())
					} else {
						ui.EventLine(ts, "", singleEvent, true, fmt.Sprintf("HTTP %d  %dms", code, latency))
					}
					fmt.Println()
					ui.Outro("Done.")
				}
				return nil
			}

			// Multi-event mode
			n, _ := cmd.Flags().GetInt("events")
			sequence := make([]string, n)
			for i := range n {
				sequence[i] = allTypes[i%len(allTypes)]
			}

			if output == "json" {
				ui.PrintJSON(map[string]any{
					"status":    "started",
					"simulated": true,
					"target":    target,
					"sessionId": sessionID,
					"dryRun":    dryRun,
					"events":    n,
				})
			} else {
				ui.Intro("riskos webhook simulate", "")
				if dryRun {
					ui.StepActive(ui.Warn.Render("Dry-run") + " — printing payloads without delivering")
				} else {
					ui.StepActive(ui.Warn.Render("Simulation mode") + " — delivering example payloads from the OpenAPI spec")
				}
				ui.StepValue("Target", target, ui.Key)
				ui.StepValue("Session", sessionID, ui.Muted)
				ui.StepValue("Events", fmt.Sprintf("%d", n), ui.Muted)
				ui.LogInfo("Payloads are synthetic examples — not real production data")
				ui.LogInfo("Ctrl+C to stop")
				ui.Bar()
				fmt.Println()
			}

			ctx := cmd.Context()
			bar := ui.NewProgressBar(len(sequence), "simulating events")

			received, forwarded, failed := 0, 0, 0
			start := time.Now()

			for _, eventType := range sequence {
				select {
				case <-ctx.Done():
					goto summary
				default:
				}
				delay := time.Duration(fwdMinDelayMs+rand.Intn(fwdMaxDelayMs-fwdMinDelayMs)) * time.Millisecond
				select {
				case <-ctx.Done():
					goto summary
				case <-time.After(delay):
				}

				payload, err := webhook.FreshPayload(examples, eventType)
				if err != nil {
					continue
				}
				received++

				if dryRun {
					forwarded++
					if output == "json" {
						ui.PrintJSON(map[string]any{
							"status":    "dry_run",
							"simulated": true,
							"event":     eventType,
							"payload":   payload,
						})
					} else {
						ts := time.Now().Format("15:04:05")
						ui.EventLine(ts, "", eventType, true, "dry-run")
						bar.Advance()
					}
					continue
				}

				code, latency, deliverErr := webhook.DeliverEvent(ctx, target, payload)
				if output == "json" {
					if deliverErr != nil {
						failed++
						ui.PrintJSON(map[string]any{
							"status":    "delivery_failed",
							"simulated": true,
							"event":     eventType,
							"delivery":  map[string]any{"httpCode": 0, "latencyMs": latency, "errMsg": deliverErr.Error()},
						})
					} else {
						forwarded++
						ui.PrintJSON(map[string]any{
							"status":    "delivered",
							"simulated": true,
							"event":     eventType,
							"delivery":  map[string]any{"httpCode": code, "latencyMs": latency},
						})
					}
				} else {
					ts := time.Now().Format("15:04:05")
					if deliverErr != nil {
						failed++
						ui.EventLine(ts, "", eventType, false, deliverErr.Error())
					} else {
						forwarded++
						ui.EventLine(ts, "", eventType, true, fmt.Sprintf("HTTP %d  %dms", code, latency))
					}
					bar.Advance()
				}
			}

		summary:
			bar.Done()
			dur := time.Since(start).Round(time.Second)

			if output == "json" {
				ui.PrintJSON(map[string]any{
					"status":    "complete",
					"simulated": true,
					"sessionId": sessionID,
					"stats": map[string]any{
						"received": received, "forwarded": forwarded,
						"failed": failed, "duration": dur.String(),
					},
				})
			} else {
				fmt.Println()
				ui.Bar()
				ui.StepDone("Simulation ended")
				ui.StepValue("Received", fmt.Sprintf("%d", received), ui.Val)
				ui.StepValue("Forwarded", fmt.Sprintf("%d", forwarded), ui.Success)
				if failed > 0 {
					ui.StepValue("Failed", fmt.Sprintf("%d", failed), ui.Err)
				} else {
					ui.StepValue("Failed", "0", ui.Muted)
				}
				ui.StepValue("Duration", dur.String(), ui.Muted)
				ui.Outro("Simulation complete.")
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
	webhookCreateCmd.Flags().String("bearer-token", "", "Token for BEARER_TOKEN auth (prefer --bearer-token-file)")
	webhookCreateCmd.Flags().String("bearer-token-file", "", "Read bearer token from file (avoids exposing in process list)")
	webhookCreateCmd.MarkFlagRequired("name")
	webhookCreateCmd.MarkFlagRequired("url")
	webhookCreateCmd.MarkFlagRequired("use-case")

	webhookUpdateCmd.Flags().String("name", "", "Existing webhook name (required)")
	webhookUpdateCmd.Flags().String("url", "", "New endpoint URL (required)")
	webhookUpdateCmd.Flags().String("use-case", "", "Use case (required)")
	webhookUpdateCmd.Flags().StringArray("workflow-events", nil, "Workflow event names")
	webhookUpdateCmd.Flags().StringArray("case-events", nil, "Case event names")
	webhookUpdateCmd.Flags().String("auth-scheme", "BEARER_TOKEN", "Auth scheme")
	webhookUpdateCmd.Flags().String("bearer-token", "", "Bearer token (prefer --bearer-token-file)")
	webhookUpdateCmd.Flags().String("bearer-token-file", "", "Read bearer token from file")
	webhookUpdateCmd.MarkFlagRequired("name")
	webhookUpdateCmd.MarkFlagRequired("url")
	webhookUpdateCmd.MarkFlagRequired("use-case")

	webhookSimulateCmd.Flags().String("target", "", "Local URL to deliver simulated events to")
	webhookSimulateCmd.Flags().Int("events", 8, "Number of events to simulate")
	webhookSimulateCmd.Flags().String("event", "", "Send a single event of this type (e.g. evaluation_completed)")
	webhookSimulateCmd.Flags().Bool("dry-run", false, "Print payloads without making HTTP requests")

	webhookCmd.AddCommand(webhookEventsCmd, webhookListCmd, webhookCreateCmd, webhookUpdateCmd, webhookSimulateCmd)
	return webhookCmd
}
