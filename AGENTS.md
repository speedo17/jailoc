# AGENTS.md

## Project Overview

`jailoc` — CLI tool that manages sandboxed Docker Compose environments for headless OpenCode coding agents. Each workspace gets isolated containers with network restrictions, privilege dropping, and bind-mounted project paths.

**Module**: `github.com/seznam/jailoc`
**Language**: Go 1.26+
**Key deps**: docker/compose/v5, docker/docker, docker/cli, BurntSushi/toml, spf13/cobra

## Architecture

```
cmd/jailoc/main.go          Entry point → cmd.Execute(version, commit, date)
internal/
  cmd/                       Cobra CLI: root.go + 7 subcommands (up, down, attach, logs, status, add, config_cmd)
  config/                    TOML config parsing, validation, mutation (~/.config/jailoc/config.toml)
  workspace/                 Workspace name → resolved paths, port assignment, CWD matching
  compose/                   docker-compose.yml generation from Go template
  docker/                    Docker Compose SDK + Engine SDK — image build/pull, compose lifecycle
  embed/                     go:embed assets (Dockerfile, compose template, entrypoint.sh, default config)
  integration_test.go        //go:build integration — end-to-end with real Docker
docs/                        MkDocs documentation (zensical theme)
```

### Data flow (`jailoc up`)

1. `config.Load` → read `~/.config/jailoc/config.toml`
2. `workspace.Resolve` → name → paths, port (`4096 + alphabetical index`), allowed hosts/networks
3. `docker.ResolveImage` → 4-step cascade: local Dockerfile → registry pull → embedded fallback → workspace layer
4. `compose.WriteCompose` → render template → `~/.cache/jailoc/{workspace}/docker-compose.yml`
5. `docker.Up` → Compose SDK starts two containers: `opencode` + `dind` sidecar

### Container architecture

Two services per workspace on an internal Docker network:

- **opencode container**: runs `opencode serve` as UID 1000 (agent), workspace paths bind-mounted (rw), OC config mounted (ro), port exposed to host
- **dind container**: privileged Docker daemon on TLS :2376, shared TLS certs + docker data via named volumes
- **entrypoint.sh**: runs as root → iptables ACCEPT for DinD/gateway/allowed hosts → DROP for RFC 1918/link-local/CGNAT → chown data dirs → `setpriv --reuid=1000 --regid=1000 --inh-caps=-all --no-new-privs`

## Conventions

### Error handling
- Always wrap: `fmt.Errorf("context: %w", err)` — never return bare `err`
- No custom error types — `fmt.Errorf` wrapping everywhere
- No logging library — `fmt.Printf` for user output, errors propagate up the stack

### Code organization
- One file per package concern: `docker.go`, `compose.go`, `workspace.go`, `config.go`
- All packages under `internal/` — nothing exported
- No `exec.Command` shellouts — everything via Go SDK
- Lazy init via `sync.Once` in docker client (`svcOnce`, `svcErr`, `svc`)

### Validation rules
- Workspace names: `^[a-z0-9-]+$`
- Forbidden mount prefixes: `/home/agent`, `/usr`, `/etc`, `/var`, `/bin`, `/sbin`, `/lib`, `/lib64`
- CIDR validation via `net.ParseCIDR`
- Path expansion: `~` → `$HOME`

### Embedded assets (`internal/embed/assets/`)
- `Dockerfile` — fallback base image when registry pull fails
- `docker-compose.yml.tmpl` — Go template for compose generation (auto-generated, do not edit manually)
- `entrypoint.sh` — container entrypoint: iptables setup → privilege drop
- `config.toml.default` — default config written on first run

## Testing

- Unit tests: `*_test.go` beside source, `t.Parallel()`, table-driven with `t.Run`
- Integration tests: `internal/integration_test.go`, build tag `//go:build integration`, `TestMain` for setup/teardown, builds the binary and runs against real Docker
- Custom `assertContains` helper — no testify
- No mocks, fixtures, or testdata directories

## CI/CD

- **Build**: `go build` with ldflags (version, commit, date)
- **Test**: `go test` + `go vet`; integration tests on `v*` tags with DinD
- **Lint**: golangci-lint v2.10.1 (gosec, staticcheck, gocritic)
- **Release**: GoReleaser on `v*` tags → Linux/Darwin × amd64/arm64, `CGO_ENABLED=0`
- **Image**: build + push base Docker image to registry on `v*` tags

## Commits

`type(scope): description` — types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`. Imperative mood.
