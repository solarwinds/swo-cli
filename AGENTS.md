# AGENTS.md

`swo` is the Go command-line client for SolarWinds Observability: `swo logs get` and `swo entities list|get|update|list-types`.

This is a public repository. Never put internal references in code, comments, commit messages, branch names, PR titles or PR descriptions: no issue-tracker keys (e.g. `ABC-123`), no internal URLs (issue tracker, wiki, chat, internal hostnames or environments), and no internal team, project or customer names.

## Layout

- `cmd/swo/` - app wiring, global flags, `version` variable.
- `config/` - loads and validates API URL and token.
- `logs/` - `logs get` command, options and API client.
- `entities/` - `entities` commands, options and API client.
- `shared/` - helpers used by several commands (e.g. slog setup).
- `scripts/sign-windows.sh` - Windows code signing hook used by GoReleaser.
- `.github/workflows/` - CI (`ci.yml`), release (`release.yml`), CodeQL.

## Commands

These match `.github/workflows/ci.yml`. A change is done when all of them pass.

```sh
go build ./cmd/swo
go test -race ./...   # CI runs this on Linux, macOS and Windows
golangci-lint run   # GOLANGCI_LINT_VERSION from the Makefile (or: make lint)
go mod tidy && git diff --exit-code -- go.mod go.sum
make vuln            # govulncheck for every release target
goreleaser check    # GORELEASER_VERSION from the Makefile, also used by release.yml
```

`make test`, `make lint` and `make vuln` run these checks locally. `make lint` builds golangci-lint from source, while CI uses the release binary of the same version, so results can differ in rare cases. Tool versions live only in the `Makefile` (`GOLANGCI_LINT_VERSION`, `GOIMPORTS_VERSION`, `GOVULNCHECK_VERSION`, `GORELEASER_VERSION`); CI reads them from there. Format imports with `make fix-lint`, which groups `github.com/solarwinds/swo-cli` imports the way lint expects.

## Config behaviour

- Precedence: flags > env (`SWO_API_URL`, `SWO_API_TOKEN`) > config file > defaults.
- Config file: only `-c <path>` or the default `~/.swo-cli.yml`; files in the current directory are never read (a warning is printed if one exists). A missing or unreadable file is ignored.
- The API URL must be `https://`; `http://` is allowed only for loopback hosts.
- Tokens never go to stdout, logs, or error messages; URL validation errors do not echo the URL.
- In tests, replace `userHomeDir` instead of setting `$HOME`, so the real `~/.swo-cli.yml` is never read.

## Testing

- In new or changed tests, use `t.Setenv`, `t.TempDir` and `t.Cleanup`. Never mutate `time.Local` or `$HOME`.
- Fake the API with `httptest` servers.
- Tests must pass under `-race` and in any `TZ`.
- Never call real SWO APIs from tests or manual checks without explicit maintainer approval.

## Code

- Use sentinel errors and wrap them with `%w`.
- Command output goes to stdout; errors and diagnostics go to stderr.
- Stay on `urfave/cli/v2`. Justify any new dependency in the PR.
- Document new flags and environment variables in `README.md`.
- Keep CLI flags, environment variables and config keys backward compatible; otherwise mark the change as breaking (`!` and `BREAKING CHANGE:`).
- `--json` output is one JSON object per line, not an array.

## Release

1. Open a PR `chore: bump version to vX.Y.Z` that bumps `version` in `cmd/swo/main.go`.
2. After it merges, tag `vX.Y.Z` on `main` and push the tag.
3. `release.yml` builds the release with GoReleaser and signs the Windows and macOS binaries.

Do not change signing/notarization config or release secrets without the maintainer.

## Commits, branches, PRs

- Conventional Commits: `type(scope): imperative summary`.
- Scope is optional; suggested scopes: `config`, `logs`, `entities`, `cli`, `release`, `deps`.
- Breaking changes: `type(scope)!: ...` plus a `BREAKING CHANGE:` footer.
- Branches: `type/short-desc`.
- No internal references anywhere (see top of this file); link work items from the issue tracker side instead.
- PRs are squash-merged and single-commit PRs land with the commit subject, so keep the commit subject equal to the PR title.
- PR descriptions: TLDR first, short, no internal links.
