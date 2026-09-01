package cli

import "context"

// tokenCreds attaches the client token to every RPC as HTTP basic auth.
// RequireTransportSecurity stops gRPC from ever sending it on a plaintext
// connection, independent of what connect decided.
type tokenCreds struct{ token string }

func (t tokenCreds) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Basic " + t.token}, nil
}

func (t tokenCreds) RequireTransportSecurity() bool { return true }
