# vaultlet — outstanding work

Status as of 2026-08-31 (`6b8fd65`). `GetSecret`, `ListSecrets` and
`DeleteSecret` are implemented and verified end to end against Bitwarden;
`PutSecret` is implemented but untested against a write-enabled token.

Ordered roughly by impact. The suggested sequence is at the bottom.

---

## 1. Functional gaps

### 1.1 WatchSecrets is unimplemented

- [ ] Not started.

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

### 1.2 Compare-and-swap

- [x] `DeleteSecret` now rejects `expected_version` with `Unimplemented`,
      matching `PutSecret` (`ad85107`). The two RPCs are consistent and no
      longer mislead callers.
- [ ] Real compare-and-swap is still unimplemented.

Real CAS needs an `expectedVersion` parameter on `SecretStore.Put` and
`.Delete`, plus `ports.ErrVersionMismatch` mapped to `ABORTED`. Bitwarden has no
native CAS, so it would be a read-compare-write with a race window — which is
worth saying out loud in the adapter rather than pretending otherwise. Until
then `--expected-version` on the CLI is a flag that can only ever fail; consider
hiding it.

### 1.3 No TLS on the server

- [ ] Not started.

`grpcserver.New` builds a bare `grpc.NewServer()` with no `grpc.Creds`, so the
server is plaintext. The client defaults to TLS against system roots. Out of the
box the two halves cannot talk, and `--insecure` is not a dev convenience but
the only working mode.

Needs cert/key paths in `config.Config`, `grpc.Creds(...)` on the server, and
`--ca` on the client for a self-signed cert.

### 1.4 No authentication, authorization or audit

- [ ] Not started.

The proto commits to `PERMISSION_DENIED` and "the principal's policy", and the
`GetSecret` comment calls it "the one place where a per-read policy check and
audit record have to happen". None of it exists: no interceptor, no principal,
no policy, no audit record.

Related: `internal/app/` is an empty directory. The gRPC handlers call
`ports.SecretStore` directly, which is fine for a pure passthrough but leaves
nowhere for policy and audit to live. This layer is the natural home for both.

### 1.5 `ports.ErrReadOnly` does not exist

- [ ] Not started.

The proto documents `FAILED_PRECONDITION` for backends that refuse writes, and
`put.go`'s help text tells users about it. Nothing defines the error or maps it.
Needed before any read-only backend is added.

---

## 2. Bugs

### 2.1 `make build` does not stamp the version

- [x] Fixed. `CLI_PKG` now points at `internal/adapters/driving/cli`
      (`3f73bf3`); `make build && ./bin/vaultlet-cli version` reports
      `vaultlet 6b8fd65 (commit 6b8fd65, built 2026-08-31T19:03:52Z)`.
      The `root.go` doc comment was corrected too.
- [ ] One stale path remains: `cmd/vaultlet-cli/main.go:3` still says
      "a thin shell around internal/adapters/cli".

### 2.2 `config.Load` discards both load errors

- [x] Fixed (`6b8fd65`). Both `k.Load` calls are now checked, with
      `errors.Is(err, fs.ErrNotExist)` tolerating a missing `vaultlet.yaml` or
      `.env`, and the env-provider error propagated.
- [x] `Config.Validate()` was added and is called from `cmd/vaultlet/main.go`
      before the backend is opened. It covers `Backend` and `Listen`.

### 2.3 The server cannot shut down cleanly

- [x] `server.go` moved off `log` onto `slog`.
- [ ] Graceful shutdown is still missing: no signal handling, no
      `GracefulStop`, and `Listen` still returns no error.
- [ ] **Regression introduced by the `slog` change.** `Listen` now logs a failed
      `net.Listen` and *carries on* to `s.grpc.Serve(listener)` with a nil
      listener, which panics. `log.Fatal` used to stop there. `Listen` should
      return an `error` and let `run()` in `cmd/vaultlet/main.go` handle it —
      which also lets the deferred `store.Close()` run.

### 2.4 `domain.Namespace.Matches` is an unfinished stub

- [x] Resolved (`6b8fd65`) by dropping the dangling comment. `key.go` now ends
      cleanly at `Contains`. Glob matching remains unimplemented, which is fine
      — nothing depends on it.

### 2.5 Every handler logs "GetSecret invoked"

- [ ] New, introduced in `e5d8406`.

`PutSecret`, `ListSecrets` and `DeleteSecret` all open with
`slog.Info("GetSecret invoked")` — a copy-paste slip that makes the access log
actively misleading. While fixing the strings, three things are worth doing at
once:

- Use `slog.InfoContext(ctx, ...)` to match the `ErrorContext` calls below them.
- Include the key or namespace as an attribute; a bare "invoked" line carries no
  information a request counter would not.
- Consider a `grpc.UnaryInterceptor` instead. One interceptor logs method,
  duration and resulting status code for every RPC, cannot drift per handler,
  and is where the audit record from §1.4 will want to live anyway.

---

## 3. Missing scaffolding

### 3.1 No tests

- [ ] Not started. Still no `_test.go` anywhere in the repo, though `make test`
      exists.

- `domain`: `ParseKey`, `ParseNamespace`, `Namespace.Contains` are pure
  functions against a documented grammar. Cheapest, highest-value tests here.
- `grpcserver`: the handlers are testable against a fake `ports.SecretStore` —
  particularly the error mapping (`ErrNotFound` → `NOT_FOUND`, `ErrEmptyValue` →
  `INVALID_ARGUMENT`, empty namespace → list everything, non-empty page token →
  rejected, `expected_version` → `UNIMPLEMENTED` on both put and delete).
- `cli`: `NewRootCmd` returns a `*cobra.Command` specifically so "tests can
  execute commands with their own args and output buffers" — a seam nothing
  currently uses.

### 3.2 Pagination is server-side unimplemented

- [ ] Not started.

`ListSecrets` ignores `page_size` and always returns one page. This is legal per
the proto ("server may return fewer than requested") and the CLI already loops
on `next_page_token`, so it can wait — but it will matter on a large org.

### 3.3 Second backend

- [ ] Not started.

`internal/adapters/driven/aws/` is an empty directory. A second backend is the
whole point of the ports design and the best proof the abstraction holds.

---

## 4. Smaller items

- [ ] `list.go` uses `cobra.ExactArgs(1)`, so the CLI cannot reach the
      empty-namespace "list everything" the server now supports.
      `cobra.MaximumNArgs(1)` opens it up.
- [ ] `resolveID` lists the entire organization on every Get, Put and Delete.
      The comment acknowledges the cost; it will bite once the vault is
      non-trivial. A short-lived name→UUID cache is the obvious mitigation.
- [ ] A permissions gap and a genuinely absent secret are indistinguishable to
      the caller — both surface as `NOT_FOUND`. An org listing that returns zero
      rows overall is far more likely to be a misconfigured machine account than
      an empty vault, and is worth logging as such in `resolveID`. (Confirmed in
      practice: a machine account with no project access reads as an empty org.)
- [ ] `ListSecretsResponse{..., NextPageToken: ""}` sets a zero value explicitly
      and can be dropped.

---

## Suggested order

1. The `Listen` nil-listener regression (§2.3) — it is a panic on a real path.
2. The handler log strings (§2.5) and the last stale doc comment (§2.1).
3. Domain and handler tests, before the surface grows further.
4. Graceful shutdown in the server (the rest of §2.3).
5. `WatchSecrets`, including the port change and the Bitwarden polling loop.
6. TLS, then the policy/audit layer in `internal/app`.
