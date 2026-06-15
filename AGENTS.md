# AGENTS.md

## Repo State
- This repo currently has only `go.mod`; there is no source tree, CLI entrypoint, README, CI, lint config, formatter config, task runner, or test setup yet.
- Module path is `github.com/f1forhelp/tailed-box-cli`; keep future internal imports under this path.
- `go.mod` declares Go `1.25.1`; do not downgrade or change it unless the task explicitly requires toolchain/version work.

## Commands
- Run `go mod tidy` after adding, removing, or changing imports/dependencies.
- Use plain Go commands for now; no Makefile, task runner, or repo-specific wrappers exist.
- `go test ./...` is the expected verification command once packages exist; in the current empty repo it has no packages to run.
