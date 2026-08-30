package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	vaultletv1 "github.com/IbiliAze/vaultlet/api/gen/vaultlet/v1"
	"github.com/IbiliAze/vaultlet/internal/domain"
	"github.com/spf13/cobra"
)

func newPutCmd(opts *options) *cobra.Command {
	var (
		value           string
		fromFile        string
		expectedVersion string
	)

	cmd := &cobra.Command{
		Use:   "put <namespace/name>",
		Short: "Create or replace a secret",
		Long: "Write a value, read from --value, --from-file, or stdin.\n\n" +
			"vaultlet is read-mostly: a backend that does not accept writes fails\n" +
			"with FAILED_PRECONDITION. Manage those secrets in the backend's UI.",
		Args: cobra.ExactArgs(1),
		Example: "  vaultlet put dev/local/DB_URL --value 'postgres://…'\n" +
			"  vaultlet put dev/local/ca.pem --from-file ./ca.pem\n" +
			"  printf %s \"$TOKEN\" | vaultlet put dev/local/TOKEN\n" +
			"  vaultlet put dev/local/DB_URL --value new --expected-version 7",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := domain.ParseKey(args[0])
			if err != nil {
				return err
			}

			raw, err := readValue(cmd, value, fromFile)
			if err != nil {
				return err
			}

			req := &vaultletv1.PutSecretRequest{Key: key.String(), Value: raw}
			// Unset means an unconditional write; only send the field when the
			// user actually asked for compare-and-swap.
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

			resp, err := client.PutSecret(ctx, req)
			if err != nil {
				return rpcError("put "+key.String(), err)
			}

			meta := toMetaJSON(resp.GetMeta())
			if opts.output == "json" {
				return writeJSON(cmd.OutOrStdout(), meta)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s version %s\n", meta.Key, meta.Version)
			return err
		},
	}

	f := cmd.Flags()
	f.StringVar(&value, "value", "", "the value itself — visible in the process list, prefer stdin")
	f.StringVar(&fromFile, "from-file", "", "read the value from this file")
	f.StringVar(&expectedVersion, "expected-version", "", "write only if the current version matches (ABORTED otherwise)")
	cmd.MarkFlagsMutuallyExclusive("value", "from-file")

	return cmd
}

// readValue resolves the value from exactly one of --value, --from-file, or
// stdin, and refuses an empty one: domain.NewSecret rejects it server-side, so
// failing here saves a round trip and gives a clearer message.
func readValue(cmd *cobra.Command, value, fromFile string) ([]byte, error) {
	var raw []byte

	switch {
	case cmd.Flags().Changed("value"):
		raw = []byte(value)
	case fromFile != "":
		b, err := os.ReadFile(fromFile)
		if err != nil {
			return nil, fmt.Errorf("read --from-file: %w", err)
		}
		raw = b
	default:
		b, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		raw = b
	}

	if len(raw) == 0 {
		return nil, errors.New("empty value: pass --value, --from-file, or pipe the value on stdin")
	}
	return raw, nil
}
