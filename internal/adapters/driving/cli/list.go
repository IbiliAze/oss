package cli

import (
	"fmt"
	"text/tabwriter"

	vaultletv1 "github.com/IbiliAze/vaultlet/api/gen/vaultlet/v1"
	"github.com/IbiliAze/vaultlet/internal/domain"
	"github.com/spf13/cobra"
)

func newListCmd(opts *options) *cobra.Command {
	var pageSize int32

	cmd := &cobra.Command{
		Use:     "list <namespace>",
		Aliases: []string{"ls"},
		Short:   "List secret metadata under a namespace",
		Long: "List every secret at or beneath a namespace. Values are never\n" +
			"returned — use `vaultlet get` for those.",
		Args:    cobra.ExactArgs(1),
		Example: "  vaultlet list payments/prod\n  vaultlet ls payments -o json",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.validateOutput(); err != nil {
				return err
			}
			ns, err := domain.ParseNamespace(args[0])
			if err != nil {
				return err
			}

			client, closeConn, err := opts.connect()
			if err != nil {
				return err
			}
			defer closeConn()

			out := cmd.OutOrStdout()
			tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			if opts.output == "text" {
				fmt.Fprintln(tw, "KEY\tVERSION\tCREATED")
			}

			// Pagination is the client's problem, not the caller's: keep asking
			// until the server stops handing back a token.
			var token string
			for {
				ctx, cancel := opts.callContext(cmd.Context())
				resp, err := client.ListSecrets(ctx, &vaultletv1.ListSecretsRequest{
					Namespace: ns.String(),
					PageSize:  pageSize,
					PageToken: token,
				})
				cancel()
				if err != nil {
					return rpcError("list "+ns.String(), err)
				}

				for _, meta := range resp.GetSecrets() {
					m := toMetaJSON(meta)
					if opts.output == "json" {
						if err := writeJSON(out, m); err != nil {
							return err
						}
						continue
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\n", m.Key, m.Version, m.CreatedAt)
				}

				token = resp.GetNextPageToken()
				if token == "" {
					break
				}
			}

			if opts.output == "text" {
				return tw.Flush()
			}
			return nil
		},
	}

	cmd.Flags().Int32Var(&pageSize, "page-size", 0, "secrets per request; 0 lets the server choose")

	return cmd
}
