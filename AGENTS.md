# AGENTS.md

This file guides AI coding agents (Claude Code, Codex, etc.) working in this repository. CLAUDE.md is a symlink to this file.

CovenantSQL is a decentralized SQL database in Go. Block producers (`cqld`) run the main chain, miners (`cql-minerd`) serve databases, each with its own SQL chain, and clients sign every query. Most of the code is consensus, replication or network code: a change can pass on one node and still make nodes disagree.

## Core Principles (CRITICAL)

**Less is more. The simplest solution is the best solution.** The action hierarchy for every change: **Delete > Replace > Add**.

1. **Solve at the owner**: Put behavior in the code path that owns or observes it. For fixes, do not use staleness checks, initialization flags, skip-first-call branches, or retries/sleeps to hide an unfixed root cause. When such a mechanism is required by the protocol or lifecycle, implement it at the owner. Relocate misplaced triggers and delete the wrong path. For features, extend the existing owner rather than creating a parallel abstraction.
2. **Search and reuse first**: Search the whole repository before creating a helper, type, or package. Reuse or adapt what exists and delete duplicate paths. Three similar lines beat a helper nobody else calls.
3. **Delete and modify existing code before creating new code**: Before adding code, consider deleting incorrect paths, modifying existing implementations, and moving behavior to its owner. Before protecting state, check who reads it: state with no reader, or a single reader that can compute it, gets deleted, not locked (#414 fixed a race by deleting counters only a log line read, instead of locking them as #412 did).
4. **Keep scope minimal**: Implement only the simplest complete solution. Avoid impossible-state handling, speculative flags, compatibility shims without a concrete compatibility requirement, and unrelated cleanup. Rely on existing coverage and focused validation; new tests should cover real behavior not already covered. For bugfixes, verify that the regression test fails without the fix and passes with it whenever feasible.
5. **Ship zero-regression changes**: Understand what you remove instead of keeping broken code as insurance. Remove unused imports, functions, types, files, and comments. Do not break existing behavior unless the PR removes it on purpose, with evidence.

**Review gate:** for every addition, check whether deleting or changing existing code would solve the problem just as completely and more simply. If so, reject the duplicate implementation.

## Consensus invariants

- **Determinism.** Every miner of a database executes the same queries and must get the same results, and block producers must build the same blocks. Nothing that reaches SQL results, blocks, or hashes may depend on the host: current time, time zone, randomness, map iteration order, local paths. Storage opened by `xenomint/sqlite.NewSqlite` uses the `nocurrenttime` VFS, and every build uses the `sqlite_omit_localtime` tag (#402); keep both. `xenomint/query_sanitizer.go` additionally rejects `'now'`/`'localtime'` literals for a clear error.
- **Hash-stable encoding.** Types marked `//go:generate hsp` (in `types/`, `proto/`, `crypto/verifier/`, `blockproducer/interfaces/`) are hashed and signed through their generated `MarshalHash`. Changing their fields changes hashes and signatures and breaks existing chains. Never edit `*_gen.go` by hand: install `hsp` with `go install github.com/CovenantSQL/HashStablePack/v2/hsp@<version in go.mod>` and run `go generate` on the package. `TestMarshalHashGolden` in `types/` pins the bytes of every generated `MarshalHash`; run it with `-update` only for an intended encoding change. Earlier encodings stay in versioned files such as `types/updatebilling_updatebillingheader_oldver_gen.go`, which hsp rewrites only while `types/updatebilling_gen.go` is the pre-versioning one from 9ebc1434.
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
- Reject compatibility shims without a concrete compatibility requirement, duplicated helpers, dead code, unrelated cleanup, and new fields or data with no reader.
- Delete tests whose assertions only validate mock setup or interactions.

## Build and test

Use `Makefile` for build targets and `.github/workflows/ci.yml` for the current Go version and CI environment. Run `make all` before `./alltest.sh` to build the binaries that node tests start.

Builds need cgo (`CGO_ENABLED=1`) and `libicu-dev` for the miner's `sqlite_icu` tag.

Export `UNITTESTTAGS` with the value from `.github/workflows/ci.yml` before running `./alltest.sh`; the script defaults to empty tags otherwise. Pass the same tags to individual package tests, e.g. `go test -race -tags "$UNITTESTTAGS" ./sqlchain/`.

The node test packages selected in `alltest.sh` start BP and miner processes from `bin/*.test` on fixed ports. Run them one package at a time (`-p 1`) and with `-count=1`, since `go test` caching cannot see the binaries change.

Run `goimports` and `go vet` on changed packages. Validate on linux/amd64; limit to 4 CPUs (e.g. `docker run --cpus 4`) to approximate the CI runner for timing-sensitive tests.

## Conventions

- New Go files start with the Apache 2.0 header of the existing files, with the current year: `Copyright <year> The CovenantSQL Authors.`
- Tests use goconvey; use `ShouldResemble` for structs and `ShouldPointTo`/`ShouldNotPointTo` for pointer identity.
- After dependency updates, run `go mod tidy` and let it determine direct and indirect requirements.
- Never edit generated files: `*_gen.go` (hsp) and `sqlchain/observer/statik/statik.go` (statik).
