package cli

import (
	"encoding/base64"
	"fmt"
	"os"

	vaultletv1 "github.com/IbiliAze/vaultlet/api/gen/vaultlet/v1"
	"github.com/IbiliAze/vaultlet/internal/domain"
	"github.com/spf13/cobra"
)

func newGetCmd(opts *options) *cobra.Command {
	var (
		outFile string
		noTrail bool
	)

	cmd := &cobra.Command{
		Use:   "get <namespace/name>",
		Short: "Read one secret value",
		Long: "Read one secret and write its value to stdout, or to --out.\n\n" +
			"This is the only command that transports secret bytes; every read is\n" +
			"policy-checked and audited server-side.",
		Args: cobra.ExactArgs(1),
		Example: "  vaultlet get payments/prod/DB_URL\n" +
			"  vaultlet get payments/prod/kubeconfig --out ./kubeconfig\n" +
			"  vaultlet get payments/prod/DB_URL -o json",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.validateOutput(); err != nil {
				return err
			}
			key, err := domain.ParseKey(args[0])
			if err != nil {
				return err
			}

			client, closeConn, err := opts.connect()
			if err != nil {
				return err
			}
			defer closeConn()

			ctx, cancel := opts.callContext(cmd.Context())
			defer cancel()

			resp, err := client.GetSecret(ctx, &vaultletv1.GetSecretRequest{Key: key.String()})
			if err != nil {
				return rpcError("get "+key.String(), err)
			}
			secret := resp.GetSecret()

			if outFile != "" {
				// 0600: the value is a secret the moment it lands on disk.
				if err := os.WriteFile(outFile, secret.GetValue(), 0o600); err != nil {
					return fmt.Errorf("write --out: %w", err)
				}
				return nil
			}

			if opts.output == "json" {
				return writeJSON(cmd.OutOrStdout(), struct {
					metaJSON
					Value string `json:"value"`
				}{
					metaJSON: toMetaJSON(secret.GetMeta()),
					// Base64: values are arbitrary bytes and need not be UTF-8.
					Value: base64.StdEncoding.EncodeToString(secret.GetValue()),
				})
			}

			w := cmd.OutOrStdout()
			if _, err := w.Write(secret.GetValue()); err != nil {
				return err
			}
			if !noTrail {
				_, err = fmt.Fprintln(w)
			}
			return err
		},
	}

	cmd.Flags().StringVar(&outFile, "out", "", "write the value to this file (mode 0600) instead of stdout")
	cmd.Flags().BoolVar(&noTrail, "no-newline", false, "do not append a trailing newline to stdout")

	return cmd
}
