# vaultlet — outstanding work

Status as of 2026-08-31. `GetSecret`, `ListSecrets` and `DeleteSecret` are
implemented and verified end to end against Bitwarden; `PutSecret` is
implemented but untested against a write-enabled token.

Ordered roughly by impact. The suggested sequence is at the bottom.

---

## 1. Functional gaps

### 1.1 WatchSecrets is unimplemented

The largest remaining piece. The proto declares the RPC and the CLI already has
a complete client for it — `internal/adapters/driving/cli/watch.go` handles the
initial snapshot, `IN_SYNC`, event rendering in both text and JSON, and Ctrl-C —
but the server answers `Unimplemented` via the embedded
`UnimplementedSecretServiceServer`.

What it needs:

- A watch method on `ports.SecretStore`. The port has no streaming surface today.
- A polling implementation in the Bitwarden adapter. Bitwarden exposes no change
  feed, so the loop diffs successive `List` calls and emits ADDED / UPDATED /
  DELETED from the difference. `bitwarden.Config.PollInterval` (`poll_interval:
  30s` in `vaultlet.yaml`) is parsed for exactly this and currently unused.
- Per the proto: emit one ADDED per existing secret, then exactly one `IN_SYNC`,
  then live changes. Metadata only — never values.

Note that `versionAt` derives the version from `RevisionDate`, so an UPDATED
event is detectable as a version change on an unchanged key.

### 1.2 Compare-and-swap is inconsistent between put and delete

`PutSecret` correctly returns `Unimplemented` when `expected_version` is set.
`DeleteSecret` accepts the same field and **silently ignores it** —
`internal/adapters/driving/grpcserver/handlers.go` never reads
`req.ExpectedVersion` — while `delete.go` advertises `--expected-version` to
users. A caller who passes it believes they performed a guarded delete and did
not.

Short-term fix, mirroring Put:

```go
if req.ExpectedVersion != nil {
    return nil, status.Error(codes.Unimplemented, "expected_version is not supported")
}
```

Long-term: real CAS needs an `expectedVersion` parameter on `SecretStore.Put`
and `.Delete`, plus `ports.ErrVersionMismatch` mapped to `ABORTED`. Bitwarden
has no native CAS, so it would be a read-compare-write with a race window —
which is worth saying out loud in the adapter rather than pretending otherwise.

### 1.3 No TLS on the server

`grpcserver.New` builds a bare `grpc.NewServer()` with no `grpc.Creds`, so the
server is plaintext. The client defaults to TLS against system roots. Out of the
box the two halves cannot talk, and `--insecure` is not a dev convenience but
the only working mode.

Needs cert/key paths in `config.Config`, `grpc.Creds(...)` on the server, and
`--ca` on the client for a self-signed cert.

### 1.4 No authentication, authorization or audit

The proto commits to `PERMISSION_DENIED` and "the principal's policy", and the
`GetSecret` comment calls it "the one place where a per-read policy check and
audit record have to happen". None of it exists: no interceptor, no principal,
no policy, no audit record.

Related: `internal/app/` is an empty directory. The gRPC handlers call
`ports.SecretStore` directly, which is fine for a pure passthrough but leaves
nowhere for policy and audit to live. This layer is the natural home for both.

### 1.5 `ports.ErrReadOnly` does not exist

The proto documents `FAILED_PRECONDITION` for backends that refuse writes, and
`put.go`'s help text tells users about it. Nothing defines the error or maps it.
Needed before any read-only backend is added.

---

## 2. Bugs

### 2.1 `make build` does not stamp the version

`Makefile` sets `CLI_PKG := github.com/IbiliAze/vaultlet/internal/adapters/cli`,
but the package moved to `.../internal/adapters/driving/cli`. The linker
silently ignores `-X` on a symbol that does not exist:

```
$ make build && ./bin/vaultlet-cli version
vaultlet dev (commit none, built unknown)
```

Fix the path in the Makefile. The same stale path appears in the doc comments in
`internal/adapters/driving/cli/root.go` and `cmd/vaultlet-cli/main.go`.

### 2.2 `config.Load` discards both load errors

`internal/config/config.go` ignores the return of both `k.Load` calls, so a
malformed `vaultlet.yaml` or a typo'd `.env` silently produces zero config — a
failure that presents as "wrong credentials" rather than "file not parsed".

A missing file does need tolerating, but via `errors.Is(err, fs.ErrNotExist)`
rather than by discarding every error. `Config` also has no `Validate()`:
`Listen` and `Backend` are unchecked (`newStore` catches an unknown backend, but
only after the config is loaded).

### 2.3 The server cannot shut down cleanly

`Server.Listen` blocks forever and calls `log.Fatal` on error, so the
`defer closer.Close()` in `cmd/vaultlet/main.go` never runs and the Bitwarden
FFI handle leaks on every exit. There is no signal handling and no
`GracefulStop`. The CLI does this properly (`signal.NotifyContext`); the server
should too.

`server.go` is also the last file using `log` rather than `slog`, and
`log.Fatal` inside an adapter denies the caller any chance to handle the error.

### 2.4 `domain.Namespace.Matches` is an unfinished stub

`internal/domain/key.go` ends at line 160 mid-sentence:
`// or "payments/**". A`. It compiles — a dangling comment is legal Go — but the
glob matching it promises does not exist. Either finish it or drop the comment.

---

## 3. Missing scaffolding

### 3.1 No tests

There is no `_test.go` anywhere in the repo, though `make test` exists.

- `domain`: `ParseKey`, `ParseNamespace`, `Namespace.Contains` are pure
  functions against a documented grammar. Cheapest, highest-value tests here.
- `grpcserver`: the handlers are testable against a fake `ports.SecretStore` —
  particularly the error mapping (`ErrNotFound` → `NOT_FOUND`, `ErrEmptyValue` →
  `INVALID_ARGUMENT`, empty namespace → list everything, non-empty page token →
  rejected).
- `cli`: `NewRootCmd` returns a `*cobra.Command` specifically so "tests can
  execute commands with their own args and output buffers" — a seam nothing
  currently uses.

### 3.2 Pagination is server-side unimplemented

`ListSecrets` ignores `page_size` and always returns one page. This is legal per
the proto ("server may return fewer than requested") and the CLI already loops
on `next_page_token`, so it can wait — but it will matter on a large org.

### 3.3 Second backend

`internal/adapters/driven/aws/` is an empty directory. A second backend is the
whole point of the ports design and the best proof the abstraction holds.

---

## 4. Smaller items

- `list.go` uses `cobra.ExactArgs(1)`, so the CLI cannot reach the empty-namespace
  "list everything" the server now supports. `cobra.MaximumNArgs(1)` opens it up.
- `resolveID` lists the entire organization on every Get, Put and Delete. The
  comment acknowledges the cost; it will bite once the vault is non-trivial.
  A short-lived name→UUID cache is the obvious mitigation.
- A permissions gap and a genuinely absent secret are indistinguishable to the
  caller — both surface as `NOT_FOUND`. An org listing that returns zero rows
  overall is far more likely to be a misconfigured machine account than an empty
  vault, and is worth logging as such in `resolveID`.
- `ListSecretsResponse{..., NextPageToken: ""}` sets a zero value explicitly and
  can be dropped.

---

## Suggested order

1. Makefile `CLI_PKG` path and the `config.Load` error handling — minutes each.
2. The `DeleteSecret` CAS inconsistency — it currently misleads users.
3. Domain and handler tests, before the surface grows further.
4. Graceful shutdown and `slog` in the server.
5. `WatchSecrets`, including the port change and the Bitwarden polling loop.
6. TLS, then the policy/audit layer in `internal/app`.
