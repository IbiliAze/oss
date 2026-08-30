package cli

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	vaultletv1 "github.com/IbiliAze/vaultlet/api/gen/vaultlet/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// connect dials the server and returns a client plus the cleanup the caller
// must defer. Dialling is lazy, so an unreachable server surfaces on the first
// RPC as UNAVAILABLE rather than here.
func (o *options) connect() (vaultletv1.SecretServiceClient, func() error, error) {
	creds, err := o.transportCredentials()
	if err != nil {
		return nil, nil, err
	}

	conn, err := grpc.NewClient(o.server, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, nil, fmt.Errorf("dial %s: %w", o.server, err)
	}
	return vaultletv1.NewSecretServiceClient(conn), conn.Close, nil
}

func (o *options) transportCredentials() (credentials.TransportCredentials, error) {
	if o.insecure {
		if o.caFile != "" {
			return nil, errors.New("--insecure and --ca are mutually exclusive")
		}
		return insecure.NewCredentials(), nil
	}
	if o.caFile == "" {
		// System roots: correct against a server with a publicly trusted cert.
		return credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS13}), nil
	}

	pem, err := os.ReadFile(o.caFile)
	if err != nil {
		return nil, fmt.Errorf("read --ca: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("read --ca: %s contains no certificates", o.caFile)
	}
	return credentials.NewTLS(&tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS13}), nil
}

// callContext bounds a unary RPC by --timeout while staying a child of the
// command's context, so Ctrl-C still cancels.
func (o *options) callContext(parent context.Context) (context.Context, context.CancelFunc) {
	if o.timeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, o.timeout)
}

// rpcError strips the "rpc error: code = ..." envelope so users see the message
// the server actually wrote. The status code stays available to callers that
// need it via status.FromError on the original error.
func rpcError(verb string, err error) error {
	if st, ok := status.FromError(err); ok {
		return fmt.Errorf("%s: %s (%s)", verb, st.Message(), st.Code())
	}
	return fmt.Errorf("%s: %w", verb, err)
}
