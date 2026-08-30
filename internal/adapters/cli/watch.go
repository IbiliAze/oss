package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	vaultletv1 "github.com/IbiliAze/vaultlet/api/gen/vaultlet/v1"
	"github.com/IbiliAze/vaultlet/internal/domain"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newWatchCmd(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch <namespace>",
		Short: "Stream changes under a namespace",
		Long: "Subscribe to a namespace and print an event whenever a secret under\n" +
			"it is added, updated or deleted.\n\n" +
			"The stream opens with one ADDED event per secret already present,\n" +
			"then a single IN_SYNC event; everything after that is a live change.\n" +
			"Events carry metadata only — read a rotated value with `vaultlet get`.\n\n" +
			"--timeout does not apply: the stream runs until the server ends it or\n" +
			"you interrupt.",
		Args:    cobra.ExactArgs(1),
		Example: "  vaultlet watch payments/prod\n  vaultlet watch payments/prod -o json",
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

			// Deliberately not opts.callContext: a watch is long-lived, and
			// bounding it by --timeout would kill it mid-subscription.
			ctx := cmd.Context()

			stream, err := client.WatchSecrets(ctx, &vaultletv1.WatchSecretsRequest{Namespace: ns.String()})
			if err != nil {
				return rpcError("watch "+ns.String(), err)
			}

			out := cmd.OutOrStdout()
			for {
				resp, err := stream.Recv()
				switch {
				case errors.Is(err, io.EOF):
					// The server closed the stream cleanly.
					return nil
				case status.Code(err) == codes.Canceled && ctx.Err() != nil:
					// Ctrl-C, or a cancelled parent context. Not a failure.
					return nil
				case err != nil:
					return rpcError("watch "+ns.String(), err)
				}

				if err := printEvent(out, opts.output, resp.GetEvent()); err != nil {
					return err
				}
			}
		},
	}

	return cmd
}

func printEvent(w io.Writer, format string, ev *vaultletv1.SecretEvent) error {
	kind := eventKind(ev.GetType())

	if format == "json" {
		return writeJSON(w, struct {
			Type string    `json:"type"`
			Meta *metaJSON `json:"meta,omitempty"`
		}{Type: kind, Meta: optionalMeta(ev.GetMeta())})
	}

	// IN_SYNC carries no meta: the snapshot is done, nothing to name.
	if ev.GetMeta() == nil {
		_, err := fmt.Fprintf(w, "%-8s\n", kind)
		return err
	}
	m := toMetaJSON(ev.GetMeta())
	_, err := fmt.Fprintf(w, "%-8s %s %s\n", kind, m.Key, m.Version)
	return err
}

func optionalMeta(m *vaultletv1.SecretMeta) *metaJSON {
	if m == nil {
		return nil
	}
	j := toMetaJSON(m)
	return &j
}

// eventKind renders SECRET_EVENT_TYPE_ADDED as "ADDED". An unknown value keeps
// its numeric form rather than being reported as UNSPECIFIED, so a client built
// against an older proto does not silently misreport a new event type.
func eventKind(t vaultletv1.SecretEventType) string {
	name, ok := vaultletv1.SecretEventType_name[int32(t)]
	if !ok {
		return fmt.Sprintf("TYPE_%d", int32(t))
	}
	return strings.TrimPrefix(name, "SECRET_EVENT_TYPE_")
}
