# CLI Reference

## Synopsis

```
jailoc [command] [flags]
```

## Global Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--workspace` | `-w` | `default` | Target workspace name. Overrides CWD-based auto-detection where applicable. |

---

## Commands

### `jailoc`

Auto-detect the workspace from the current working directory. If no matching workspace is found, prompts to add the current directory. If the workspace is not running, starts it. Attaches to the running environment.

```
jailoc [flags]
```

| Flag | Description |
|------|-------------|
| `--remote` | Force remote connection mode (runs `opencode attach` on the host). |
| `--exec` | Force exec connection mode (runs `docker exec` into the container). |
| `--dir` | Directory to open in opencode (forwarded as `--dir` to `opencode attach`). |

When run from a subdirectory of a workspace path, jailoc resolves the current working directory and passes it as `--dir` to `opencode attach`, so the agent opens that directory instead of the workspace root.

When neither `--remote` nor `--exec` is specified, the connection mode is determined by the workspace `mode` field in configuration, falling back to auto-detection (checks for `opencode`, then `opencode-cli` on PATH). See the [access modes how-to](../how-to/access-modes.md) for configuration steps, or the [access modes explanation](../explanation/access-modes.md) for the difference between modes.

*Note: Starting a workspace requires `OPENCODE_SERVER_PASSWORD`. See [`jailoc up`](#jailoc-up) for details.*

---

### `jailoc up`

Start the Docker Compose environment for the target workspace. No-op if the workspace containers are already running.

```
jailoc up [flags]
```

!!! warning "OPENCODE_SERVER_PASSWORD required"

    `jailoc up` requires `OPENCODE_SERVER_PASSWORD` to be set in the environment.
    An empty password leaves the OpenCode server unauthenticated.
    See [Getting Started](../tutorials/getting-started.md) for setup instructions.

    If the workspace is already running but was started without a password, `jailoc up` prints a warning with restart instructions.

Resolves the container image (see [Image Resolution](image-resolution.md)), generates a `docker-compose.yml` in `~/.cache/jailoc/{workspace}/`, and starts two containers: `opencode` and `dind`.

When `--workspace` is not set, resolves the workspace whose configured path best matches the current working directory (longest prefix). Falls back to `default`. See the [workspace configuration how-to](../how-to/workspace-configuration.md) for the full resolution order.

---

### `jailoc down`

Stop and remove the containers for the target workspace.

```
jailoc down [flags]
```

Equivalent to `docker compose down` on the generated compose file. Does not remove named volumes or the image.

When `--workspace` is not set, resolves the workspace whose configured path best matches the current working directory (longest prefix). Falls back to `default`. See the [workspace configuration how-to](../how-to/workspace-configuration.md) for the full resolution order.

---

### `jailoc status`

Print the state and assigned port for each configured workspace.

```
jailoc status [flags]
```

Output lists all workspaces defined in configuration. For each workspace, shows the container state (running, stopped, or unknown) and the host port it is assigned.

When `--workspace` is not set, resolves the workspace whose configured path best matches the current working directory (longest prefix). Falls back to `default`. See the [workspace configuration how-to](../how-to/workspace-configuration.md) for the full resolution order.

---

### `jailoc logs`

Stream container logs from the target workspace environment.

```
jailoc logs [flags]
```

Streams combined stdout and stderr from the `opencode` and `dind` containers. Follows log output until interrupted.

When no positional workspace argument and `--workspace` is not set, resolves the workspace from the current working directory (longest prefix). Falls back to `default`. See the [workspace configuration how-to](../how-to/workspace-configuration.md) for the full resolution order.

---

### `jailoc config`

Print the current resolved configuration.

```
jailoc config [flags]
```

Reads `~/.config/jailoc/config.toml`, resolves all defaults and `~` expansions, and prints the result. Useful for verifying what values are in effect.

---

### `jailoc add`

Add the current working directory to the target workspace's `paths` list.

```
jailoc add [flags]
```

Appends the current directory to `workspaces.<name>.paths` in `~/.config/jailoc/config.toml`. The path must not be under a forbidden system prefix. See the [configuration reference](configuration.md) for path validation rules.

When `--workspace` is not set, resolves the workspace from the path being added (longest prefix). If `--workspace` is set explicitly and the path is not under any of that workspace's configured paths, `jailoc add` returns an error. See the [workspace configuration how-to](../how-to/workspace-configuration.md) for the full resolution order.

*Note: If the workspace is already running, `jailoc add` restarts it to apply the new mount. This requires `OPENCODE_SERVER_PASSWORD`. See [`jailoc up`](#jailoc-up) for details.*
