package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"riskos/internal/api"
	"riskos/internal/config"
	"riskos/internal/ui"
)

// defaultStoreFn returns the default config.Store. Overridable in tests.
var defaultStoreFn = func() *config.Store {
	return config.DefaultStore()
}

// newClientFn is used by newClient for tests to inject a mock.
// The variadic interface{} is not actually used — it exists to keep
// the signature consistent without importing cobra here.
var newClientFn func(...interface{}) api.Client

var rootCmd = buildRootCmd()

// buildRootCmd constructs a fresh root command with all subcommands attached.
// It is called once at package init (stored in rootCmd) and can be called
// again in tests to get an isolated tree.
func buildRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "riskos",
		Short: "RiskOS CLI — execution layer for RiskOS integrations",
		Long: `RiskOS CLI — execution layer for RiskOS integrations.

Examples:
  riskos auth login --api-key <your-riskos-api-key>
  riskos usecases list
  riskos asset guide consumer_onboarding ./guide.md
  riskos webhook events
  riskos webhook forward --target http://localhost:3000/webhooks/riskos
  riskos testcase run Valid_Consumer_Accept`,
		Version:       "1.0.0",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	initRootFlags(root)
	attachSubcommands(root)
	return root
}

func initRootFlags(root *cobra.Command) {
	root.PersistentFlags().BoolP("quiet", "q", false,
		"Suppress informational output. Errors still print to stderr.")
	root.PersistentFlags().StringP("output", "o", "",
		"Output format: plain or json (default: plain in TTY, json when piped)")
	root.PersistentFlags().Bool("prod", false,
		"Use the production RiskOS endpoint instead of sandbox")
}

func attachSubcommands(root *cobra.Command) {
	root.AddCommand(buildAuthCmd())
	root.AddCommand(buildUsecasesCmd())
	root.AddCommand(buildWorkflowsCmd())
	root.AddCommand(buildAssetCmd())
	root.AddCommand(buildWebhookCmd())
	root.AddCommand(buildTestcaseCmd())
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ui.Abort(err.Error(), "", "")
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
func requireAuth(store ...*config.Store) *config.Credentials {
	s := defaultStoreFn()
	if len(store) > 0 && store[0] != nil {
		s = store[0]
	}
	creds, err := s.Load()
	if err != nil || creds == nil {
		ui.Abort(
			"Not authenticated.",
			"Run:  riskos auth login --api-key <key>\n  │  Or set the RISKOS_API_KEY env var",
			"",
		)
	}
	return creds
}

// newClient builds a RiskOS API client from stored credentials.
func newClient(cmd *cobra.Command, store ...*config.Store) api.Client {
	// Allow tests to inject a mock client
	if newClientFn != nil {
		return newClientFn()
	}
	s := defaultStoreFn()
	if len(store) > 0 && store[0] != nil {
		s = store[0]
	}
	creds, err := s.Load()
	if err != nil || creds == nil {
		ui.Abort(
			"Not authenticated.",
			"Run:  riskos auth login --api-key <key>\n  │  Or set the RISKOS_API_KEY env var",
			"",
		)
	}
	prod, _ := cmd.Root().PersistentFlags().GetBool("prod")
	if p, _ := cmd.Flags().GetBool("prod"); p {
		prod = p
	}
	creds.Prod = prod
	return api.NewClientFromCreds(creds)
}

// resolveOutput returns the effective output format.
func resolveOutput(cmd *cobra.Command) string {
	flag, _ := cmd.Flags().GetString("output")
	if flag == "" {
		if cmd.Root() != nil {
			flag, _ = cmd.Root().PersistentFlags().GetString("output")
		}
	}
	return ui.ResolveOutput(flag)
}

// isQuiet returns true if -q was passed or RISKOS_QUIET=1.
func isQuiet(cmd *cobra.Command) bool {
	q, _ := cmd.Flags().GetBool("quiet")
	return q || os.Getenv("RISKOS_QUIET") == "1"
}
