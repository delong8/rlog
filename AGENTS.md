# AGENTS.md

## Project Shape
- This is a single-package Go module, not a monorepo. The module path is `github.com/delong8/rlog` and `go.mod` targets Go 1.21.
- The public API lives in the root package `rlog` (`rlog.go`): `Config`, `Logger`, global `Info`/`Error`, scoped loggers from `New`, `Init`, and `Rules`.
- Logging rules are package globals. `Init` is effectively one-time after non-nil config, and `Info` starts a polling goroutine via `Init(nil)` if needed.

## Commands
- Run all tests with `go test` from the repo root; `go test ./...` is equivalent today because there is only one package.
- Run a focused test with `go test -run TestGlobalInfoUsesDefaultRulesAfterLazyInit`, `go test -run TestDefaultReadParsesRuleFile`, or `go test -run TestDefaultReadHandlesStatErrorsWithoutPanic`.
- No Makefile, CI workflow, lint config, formatter config, or task runner is present; use direct Go toolchain commands such as `go build ./...`, `go vet ./...`, and `gofmt -l .`.

## Test And Runtime Gotchas
- Keep the root file named `rlog`. It is the default runtime rules file (`./rlog`) and currently contains `*`, which enables Info output by default.
- Tests in `rlog_test.go` are assertion-based characterization and regression tests covering lazy init, scoped rules, default rule-file parsing, deadlock prevention, defensive copies, and concurrent polling.
- Tests reset package-global state through helpers, so changes to `Init`, `inited`, `configed`, `rules`, `defaultRules`, `ruleFile`, `readRules`, `pollingStarted`, or `output` can affect later tests.
- `go test -race -count=1 ./...` now passes after the global rule synchronization work and is usable as a race gate.

## API Gotchas
- `Config` currently has `File string`, `Read func() []string`, `Interval time.Duration`, `Output func(v ...any)`, and `DefaultRules []string`; use `rlog.go` as the source of truth if README examples drift.
