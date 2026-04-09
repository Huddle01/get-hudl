# hudl

The official toolchain for [Huddle01 Cloud](https://console.huddle01.com).

## Packages

| Package | Description | Status |
|---|---|---|
| [`cli/`](cli/) | Command-line interface | Available |
| [`get/`](get/) | Installer site ([get.huddle01.com](https://get.huddle01.com)) | Available |
| `mcp/` | Model Context Protocol server | Coming soon |
| `skills/` | Agent skills | Coming soon |

## Install the CLI

**macOS / Linux:**

```sh
curl -fsSL https://get.huddle01.com/hudl | sh
```

**Homebrew:**

```sh
brew install huddle01/tap/hudl
```

**Windows (PowerShell):**

```powershell
irm https://get.huddle01.com/hudl.ps1 | iex
```

**From source:**

```sh
go install github.com/Huddle01/hudl/cli/cmd/hudl@latest
```

## Quick start

```sh
hudl auth login
hudl context set --workspace my-org --region eu2
hudl vm list
hudl vm create --name web-1 --flavor m1.small --image ubuntu-24.04
hudl gpu deploy --image nvidia/cuda --gpu a100
```

## Commands

| Command | Description |
|---|---|
| `hudl auth login` | Authenticate with Huddle01 Cloud |
| `hudl context set` | Set active workspace and region |
| `hudl vm` | Manage virtual machines |
| `hudl volume` | Manage block storage volumes |
| `hudl floating-ip` | Manage floating IPs |
| `hudl network` | Manage private networks |
| `hudl security-group` | Manage security groups |
| `hudl key` | Manage SSH keys |
| `hudl gpu` | Manage GPU deployments, offers, images, and API keys |
| `hudl flavor list` | List available instance flavors |
| `hudl image list` | List available OS images |
| `hudl region list` | List available regions |

Run `hudl --help` or `hudl <command> --help` for full usage.

## Configuration

Stored in `~/.hudl/config.toml`:

```toml
[auth]
token = "..."

[context]
workspace = "my-org"
region = "eu2"
```

Override with flags (`--workspace`, `--region`) or environment variables (`HUDL_WORKSPACE`, `HUDL_REGION`).

## Development

**Prerequisites:** Go 1.25+, `make`

```sh
make build          # Build CLI binary
make dev            # Build and run
make test           # Run tests
make dist           # Cross-compile all platforms
make version        # Show current version & next suggestions
make release        # Interactive release (build, tag, GitHub release)
```

### Deploying the installer site

```sh
cd get
make deploy         # Deploy to Vercel production
```

### Release flow

```
$ make release

Current version: v0.1.0

Suggestions:
    patch  →  v0.1.1
    minor  →  v0.2.0

New version [v0.1.1]:
```

Builds all platforms, creates git tag, uploads binaries to GitHub Releases.

## Project structure

```
.
├── cli/                       # CLI
│   ├── cmd/hudl/              #   Entrypoint
│   └── internal/
│       ├── cli/               #   Cobra commands
│       ├── config/            #   Config file management
│       └── runtime/           #   HTTP client, I/O, app context
├── get/                       # get.huddle01.com
│   └── static/
│       ├── index.html         #   Landing page
│       └── install.sh         #   Shell installer
├── mcp/                       # MCP server (coming soon)
├── skills/                    # Agent skills (coming soon)
├── Makefile
└── go.mod
```

## Links

- [Documentation](https://console.huddle01.com/docs/cli)
- [Installer](https://get.huddle01.com)
- [Console](https://console.huddle01.com)

## License

Proprietary. Copyright Huddle01.
