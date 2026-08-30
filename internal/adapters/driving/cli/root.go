// Package cli is the driving adapter for the vaultlet command line client.
//
// It owns nothing but presentation: flag parsing, dialling the server, and
// rendering responses. Every command talks to the same gRPC surface an
// arbitrary client would use — there is no back door into internal/app.
package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// Build metadata, set by the linker:
//
//	go build -ldflags "-X github.com/IbiliAze/vaultlet/internal/adapters/cli.version=v0.1.0"
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// options carries the flags every command shares. One instance is built by
// NewRootCmd and handed to each subcommand, so no command reads global state.
type options struct {
	server   string
	timeout  time.Duration
	insecure bool
	caFile   string
	output   string
}

// NewRootCmd builds the command tree. It returns a *cobra.Command rather than
// running it so tests can execute commands with their own args and output
// buffers.
func NewRootCmd() *cobra.Command {
	opts := &options{}

	cmd := &cobra.Command{
		Use:   "vaultlet",
		Short: "Client for the vaultlet secrets control plane",
		Long: "vaultlet is the client for a vaultlet server: read, watch and\n" +
			"(where the backend allows it) write secrets over one access model,\n" +
			"whichever backend is behind it.",
		SilenceUsage:  true, // a runtime error is not a usage error
		SilenceErrors: true, // main prints the error once, to stderr
		// Nothing here dials or does I/O: `--help` must work with no server.
	}

	f := cmd.PersistentFlags()
	f.StringVar(&opts.server, "server", "localhost:50051", "address of the vaultlet server")
	f.DurationVar(&opts.timeout, "timeout", 10*time.Second, "per-request timeout (watch is exempt)")
	f.BoolVar(&opts.insecure, "insecure", false, "disable TLS — development only")
	f.StringVar(&opts.caFile, "ca", "", "PEM bundle used to verify the server certificate")
	f.StringVarP(&opts.output, "output", "o", "text", "output format: text|json")

	cmd.AddCommand(
		newGetCmd(opts),
		newPutCmd(opts),
		newListCmd(opts),
		newDeleteCmd(opts),
		newWatchCmd(opts),
		newVersionCmd(),
	)

	return cmd
}

// Execute runs the command tree. ctx is propagated to every RPC, so a SIGINT
// caught in main cancels an in-flight call and an open watch stream alike.
func Execute(ctx context.Context) error {
	return NewRootCmd().ExecuteContext(ctx)
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, commit and build date",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "vaultlet %s (commit %s, built %s)\n", version, commit, date)
			return err
		},
	}
}
