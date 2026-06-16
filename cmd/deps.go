package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/effectivai/riskos-cli/internal/api"
	"github.com/effectivai/riskos-cli/internal/config"
)

type depsKey struct{}

// Deps holds injectable dependencies for command execution.
type Deps struct {
	Store    *config.Store
	ClientFn func(cmd *cobra.Command, creds *config.Credentials) api.Client
}

// DefaultDeps returns production dependencies.
func DefaultDeps() *Deps {
	return &Deps{
		Store: config.DefaultStore(),
		ClientFn: func(cmd *cobra.Command, creds *config.Credentials) api.Client {
			if baseURL, _ := cmd.Root().PersistentFlags().GetString("base-url"); baseURL != "" {
				creds.BaseURL = baseURL
			}
			return api.NewClientFromCreds(creds)
		},
	}
}

func withDeps(ctx context.Context, d *Deps) context.Context {
	return context.WithValue(ctx, depsKey{}, d)
}

func depsFrom(cmd *cobra.Command) *Deps {
	if d, ok := cmd.Context().Value(depsKey{}).(*Deps); ok && d != nil {
		return d
	}
	return DefaultDeps()
}
