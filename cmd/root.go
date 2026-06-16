package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"riskos/internal/api"
	"riskos/internal/config"
	"riskos/internal/debug"
	"riskos/internal/ui"
)

var rootCmd = buildRootCmd(DefaultDeps())

// SetVersion overrides the version string (called from main with ldflags value).
func SetVersion(v string) {
	rootCmd.Version = v
}

// buildRootCmd constructs a fresh root command with all subcommands attached.
// It is called once at package init (stored in rootCmd) and can be called
// again in tests to get an isolated tree with custom deps.
func buildRootCmd(d *Deps) *cobra.Command {
	root := &cobra.Command{
		Use:   "riskos",
		Short: "RiskOS CLI — execution layer for RiskOS integrations",
		Long: `RiskOS CLI — execution layer for RiskOS integrations.

Examples:
  riskos auth login
  riskos usecase list
  riskos asset guide consumer_onboarding ./guide.md
  riskos webhook events
  riskos webhook simulate --target http://localhost:3000/webhooks/riskos
  riskos testcase run Valid_Consumer_Accept`,
		Version:       "dev",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if v, _ := cmd.Root().PersistentFlags().GetBool("debug"); v {
				debug.Enable()
			}
		},
	}
	initRootFlags(root)
	attachSubcommands(root)
	root.SetContext(withDeps(context.Background(), d))
	return root
}

func initRootFlags(root *cobra.Command) {
	root.PersistentFlags().BoolP("quiet", "q", false,
		"Suppress informational output. Errors still print to stderr.")
	root.PersistentFlags().StringP("output", "o", "",
		"Output format: plain or json (default: plain in TTY, json when piped)")
	root.PersistentFlags().String("base-url", "",
		"RiskOS API base URL (default: https://riskos.sandbox.socure.com)")
	root.PersistentFlags().Bool("debug", false,
		"Print HTTP request/response details for troubleshooting")
}

func attachSubcommands(root *cobra.Command) {
	root.AddCommand(buildAuthCmd())
	root.AddCommand(buildUsecaseCmd())
	root.AddCommand(buildWorkflowCmd())
	root.AddCommand(buildAssetCmd())
	root.AddCommand(buildWebhookCmd())
	root.AddCommand(buildTestcaseCmd())
}

func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	rootCmd.SetContext(withDeps(ctx, DefaultDeps()))

	code := 0
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(*ui.AbortError); ok {
				code = 1
			} else {
				panic(r)
			}
		}
		os.Exit(code)
	}()
	if err := rootCmd.Execute(); err != nil {
		ui.PrintAbort(err.Error(), "", "")
		code = 1
	}
}

// cmdError is a clean error type for user-facing command errors.
type cmdError struct {
	msg  string
	hint string
}

func (e *cmdError) Error() string {
	if e.hint != "" {
		return fmt.Sprintf("%s\n  hint: %s", e.msg, e.hint)
	}
	return e.msg
}

// requireAuth loads credentials or aborts with a helpful message.
func requireAuth(cmd *cobra.Command) *config.Credentials {
	d := depsFrom(cmd)
	creds, err := d.Store.Load()
	if err != nil || creds == nil {
		ui.Abort(
			"Not authenticated.",
			"Run:  riskos auth login --api-key <key>\n  │  Or set the RISKOS_API_KEY env var",
			"",
		)
	}
	return creds
}

// newClient builds a RiskOS API client using deps from context.
func newClient(cmd *cobra.Command) api.Client {
	d := depsFrom(cmd)
	creds := requireAuth(cmd)
	return d.ClientFn(cmd, creds)
}

// resolveOutput returns the effective output format.
func resolveOutput(cmd *cobra.Command) string {
	flag, _ := cmd.Flags().GetString("output")
	if flag == "" {
		if cmd.Root() != nil {
			flag, _ = cmd.Root().PersistentFlags().GetString("output")
		}
	}
	out, err := ui.ResolveOutput(flag)
	if err != nil {
		ui.Abort(err.Error(), "Valid values: plain, json", "")
	}
	return out
}

// isQuiet returns true if -q was passed or RISKOS_QUIET=1.
func isQuiet(cmd *cobra.Command) bool {
	q, _ := cmd.Flags().GetBool("quiet")
	return q || os.Getenv("RISKOS_QUIET") == "1"
}
