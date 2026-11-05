# AGENTS.md

This file guides AI coding agents (Claude Code, Codex, etc.) working in this repository. CLAUDE.md is a symlink to this file.

CovenantSQL is a decentralized SQL database in Go. Block producers (`cqld`) run the main chain, miners (`cql-minerd`) serve databases, each with its own SQL chain, and clients sign every query. Most of the code is consensus, replication or network code: a change can pass on one node and still make nodes disagree.

## Core Principles (CRITICAL)

**Less is more. The simplest solution is the best solution.** The action hierarchy for every change: **Delete > Replace > Add**.

1. **Solve at the owner**: Put behavior in the code path that owns or observes it. For fixes, never guard a symptom with a staleness check, initialization flag, skip-first-call branch, or a retry/sleep around broken logic; relocate the trigger and delete the wrong path. For features, extend the existing owner rather than creating a parallel abstraction.
2. **Search and reuse first**: Search the whole repository before creating a helper, type, or package. Reuse or adapt what exists and delete duplicate paths. Three similar lines beat a helper nobody else calls.
3. **Delete and modify existing code before creating new code**: Bugfixes are net-negative by default unless deletion and relocation are demonstrably impossible. Before protecting state, check who reads it: state with no reader, or a single reader that can compute it, gets deleted, not locked (#414 fixed a race by deleting counters only a log line read, instead of locking them as #412 did).
4. **Keep scope minimal**: Implement only the simplest complete solution. Avoid impossible-state handling, speculative flags, compatibility shims, and unrelated cleanup. Rely on existing coverage and focused validation; add test code only for an uncovered, high-risk path (races, consensus, crypto, serialization), and show that it fails without the fix.
5. **Ship zero-regression changes**: Understand what you remove instead of keeping broken code as insurance. Remove unused imports, functions, types, files, and comments. Do not break existing behavior unless the PR removes it on purpose, with evidence.

**Review gate:** for every addition, the reviewer decides whether deleting or changing existing code would have fixed the problem instead — if it would, that is a blocking finding.

## Consensus invariants

- **Determinism.** Every miner of a database executes the same queries and must get the same results, and block producers must build the same blocks. Nothing that reaches SQL results, blocks, or hashes may depend on the host: current time, time zone, randomness, map iteration order, local paths. Storage opened by `xenomint/sqlite.NewSqlite` uses the `nocurrenttime` VFS, and every build uses the `sqlite_omit_localtime` tag (#402); keep both. `xenomint/query_sanitizer.go` additionally rejects `'now'`/`'localtime'` literals for a clear error.
- **Hash-stable encoding.** Types marked `//go:generate hsp` (in `types/`, `proto/`, `crypto/verifier/`, `blockproducer/interfaces/`, `xenomint/types/`) are hashed and signed through their generated `MarshalHash`. Changing their fields changes hashes and signatures and breaks existing chains. Never edit `*_gen.go` by hand: build `hsp` from `github.com/CovenantSQL/HashStablePack/hsp` at the version in `go.mod` and run `go generate` on the package (`genMarshalHash.sh` predates modules). Earlier encodings stay in versioned files such as `types/updatebilling_updatebillingheader_oldver_gen.go`.
- **Signatures.** `crypto/asymmetric` signs and verifies with libsecp256k1 in `crypto/secp256k1`; signatures must stay byte-identical, which `TestSign` pins. `libsecp256k1/` holds upstream files unmodified, `prefix.h` renames their symbols, and `TestLink` checks the renaming. The package comment in `secp256.go` says how to update it.

## Git and PRs

- `develop` is the default branch; `master` holds only the initial commit. Never push to `develop`: it takes a PR with an approving review and a green `test` check. Never force push.
- Always start work in a new git worktree (`git worktree add`) on a feature branch named by concern (`fix/…`, `deps/…`, `chore/…`, `ci/…`, `crypto/…`). Never edit the primary checkout directly; it may hold in-flight work.
- One concern per PR. Every commit builds and passes the tests on its own.
- Agents never merge PRs; merging is the maintainer's action.
- When two approaches compete, open each as its own PR, start both descriptions with "An alternative to #N, which …. Only one of the two should be merged.", and leave the choice to the maintainer (#410/#411, #412/#414).
- `git pull --rebase` before pushing; never reset or revert commits you did not author.
- After a PR merges, remove its worktree and branch, then `git checkout develop && git pull`.

**Commits:** imperative, sentence-case subject with no type prefix and no trailing period ("Count the ack index for its stats instead of keeping counters"). The body, wrapped at 72 columns, says what was wrong and why this change is the fix.

**PR descriptions:** prose that states the problem with its evidence (the failing CI run, measured numbers), what changed, and what was deliberately left out and why. End with a `### Testing` section giving the platform, Go version and CPU limit, the exact commands that passed, and, for a reproduced bug, before/after results ("failed 20 of 20 runs on develop, 0 of 20 on this branch").

## PR Review

- Require a reproducible bug or a broadly useful feature; close speculative changes and AI-generated bloat.
- Review the full live diff independently; approvals, descriptions, and green CI are supporting evidence, not proof. A race can pass CI: rerun suspect tests with `-race` in fresh processes.
- Reject compatibility shims, duplicated helpers, dead code, unrelated cleanup, and new fields or data with no reader.
- Delete tests whose assertions only validate mock setup or interactions.

## Commands

```bash
make -j"$(nproc)" all   # bin/cqld, bin/cql-minerd, bin/cql*, and the bin/*.test binaries the node tests start
./alltest.sh            # the CI test suite; run `make all` first
make build-release      # release binaries only
make docker             # static Alpine binaries in the covenantsql/covenantsql image
./fmtGoImports.sh       # goimports -local github.com/CovenantSQL/CovenantSQL over the tree
./cleanupDB.sh          # remove *.db, *.ldb, and keystores left behind by interrupted node tests
```

Builds need cgo (`CGO_ENABLED=1`) and `libicu-dev` for the miner's `sqlite_icu` tag.

Run a single package with the CI tags, e.g. `go test -race -tags "$UNITTESTTAGS" ./sqlchain/` with `UNITTESTTAGS` from `.github/workflows/ci.yml`, so the package builds as it does in CI.

`cmd/cqld`, `cmd/cql-minerd`, `cmd/cql-fuse`, `rpc/mux`, `sqlchain/mirror`, and `sqlchain/observer` start BP and miner processes from the `bin/*.test` binaries (built with the `testbinary` tag), on fixed ports. Run them one package at a time (`-p 1`) and with `-count=1`, since `go test` caching cannot see the binaries change.

CI (`.github/workflows/ci.yml`) is one job on `ubuntu-latest` with Go 1.27: `make all`, then `./alltest.sh`, with a 90-minute timeout. It runs no linter and no `go vet`, so run `goimports` and `go vet` on the changed packages yourself. Validate on linux/amd64; limit to 4 CPUs (e.g. `docker run --cpus 4`) to approximate the CI runner for timing-sensitive tests.

Dependabot proposes Go module, Actions, and Docker updates monthly, minor and patch updates grouped. Its `go.mod` edits can move indirect requirements to the direct block; run `go mod tidy` and keep what it says.

## Where to look

- Main chain and block producer → `blockproducer/`, `cmd/cqld/`; chain transactions and blocks → `types/`
- Miner database service → `worker/`, `cmd/cql-minerd/`; per-database SQL chain → `sqlchain/` (`observer/` and `mirror/` follow chains)
- SQLite state and eventual consistency (Xenomint) → `xenomint/`; strong consistency (Kayak, BFT-Raft) → `kayak/`, `twopc/`
- Network: RPC → `rpc/`; node connections over ETLS → `naconn/`; DHT and routing → `route/`, `consistent/`; RPC structs → `proto/`
- Keys, signatures, hashing → `crypto/`
- Go `database/sql` driver → `client/`; CLI, FUSE, MySQL adapter, HTTP proxy → `cmd/cql*`
- Configuration → `conf/`; multi-node test configs → `test/` (`test/GNTE` is a network-emulator submodule)

## Conventions

- New Go files start with the Apache 2.0 header of the existing files, with the current year: `Copyright <year> The CovenantSQL Authors.`
- Tests use goconvey. Since v1.8.1 `ShouldEqual` does not convert between types and compares same-typed values with `reflect.DeepEqual`; use `ShouldResemble` for structs and `ShouldPointTo`/`ShouldNotPointTo` for pointer identity.
- Never edit generated files: `*_gen.go` (hsp) and `sqlchain/observer/statik/statik.go` (statik).
