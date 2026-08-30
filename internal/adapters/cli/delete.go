package cli

import (
	"fmt"

	vaultletv1 "github.com/IbiliAze/vaultlet/api/gen/vaultlet/v1"
	"github.com/IbiliAze/vaultlet/internal/domain"
	"github.com/spf13/cobra"
)

func newDeleteCmd(opts *options) *cobra.Command {
	var expectedVersion string

	cmd := &cobra.Command{
		Use:     "delete <namespace/name>",
		Aliases: []string{"rm"},
		Short:   "Delete a secret",
		Long: "Delete a secret. Deleting a key that does not exist is NOT_FOUND\n" +
			"rather than a silent success, so callers can tell the two apart.",
		Args:    cobra.ExactArgs(1),
		Example: "  vaultlet delete dev/local/DB_URL",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := domain.ParseKey(args[0])
			if err != nil {
				return err
			}

			req := &vaultletv1.DeleteSecretRequest{Key: key.String()}
			if cmd.Flags().Changed("expected-version") {
				req.ExpectedVersion = &expectedVersion
			}

			client, closeConn, err := opts.connect()
			if err != nil {
				return err
			}
			defer closeConn()

			ctx, cancel := opts.callContext(cmd.Context())
			defer cancel()

			if _, err := client.DeleteSecret(ctx, req); err != nil {
				return rpcError("delete "+key.String(), err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "deleted %s\n", key)
			return err
		},
	}

	cmd.Flags().StringVar(&expectedVersion, "expected-version", "", "delete only if the current version matches (ABORTED otherwise)")

	return cmd
}
