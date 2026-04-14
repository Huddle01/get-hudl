<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="get/static/logo.png" />
    <source media="(prefers-color-scheme: light)" srcset="get/static/logo_light.png" />
    <img alt="Huddle01 Cloud" src="get/static/logo_light.png" width="80" />
  </picture>
</p>

<h1 align="center">get-hudl</h1>

<p align="center">
  The official toolchain for <a href="https://console.huddle01.com">Huddle01 Cloud</a> — CLI, installer, MCP server, and agent skills.
</p>

<p align="center">
  <a href="https://get.huddle01.com"><strong>Install</strong></a> · <a href="https://console.huddle01.com/docs/cli"><strong>Docs</strong></a> · <a href="https://console.huddle01.com"><strong>Console</strong></a>
</p>

---

## Packages

| Package | Description | Status |
|---|---|---|
| [`cli/`](cli/) | Command-line interface | Available |
| [`get/`](get/) | Installer site ([get.huddle01.com](https://get.huddle01.com)) | Available |
| [`mcp/`](mcp/) | Model Context Protocol server | Available |
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
go install github.com/Huddle01/get-hudl/cli/cmd/hudl@latest
```

## Quick start

```sh
hudl login --token <cloud-key> --gpu-token <gpu-key>
hudl context set --workspace my-org --region eu2
hudl vm list
hudl vm create --name web-1 --flavor m1.small --image ubuntu-24.04
hudl gpu deploy --image nvidia/cuda --gpu a100
```

## Commands

| Command | Description |
|---|---|
| `hudl login --token <cloud-key> --gpu-token <gpu-key>` | Authenticate with Huddle01 Cloud |
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

## MCP Server

The `hudl-mcp` binary is a production-grade [Model Context Protocol](https://modelcontextprotocol.io) server that exposes every Huddle01 Cloud operation as an MCP tool. It communicates over stdio and works with any MCP-compatible client (Claude Desktop, Cursor, Claude Code, etc.).

### Build

```sh
make build-mcp      # Build MCP server binary
make build-all      # Build both CLI and MCP server
```

### Configure

Add to your MCP client config (e.g. `claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "huddle01": {
      "command": "/path/to/hudl-mcp"
    }
  }
}
```

The MCP server reads the same `~/.hudl/config.toml` and environment variables as the CLI. Authenticate first with `hudl login --token <cloud-key>` and/or `hudl login --gpu-token <gpu-key>`.

### Available Tools (60+)

| Category | Tools |
|---|---|
| **Auth** | `hudl_login`, `hudl_auth_status`, `hudl_auth_clear` |
| **Context** | `hudl_ctx_show`, `hudl_ctx_use`, `hudl_ctx_region` |
| **VMs** | `hudl_vm_list`, `hudl_vm_get`, `hudl_vm_create`, `hudl_vm_delete`, `hudl_vm_status`, `hudl_vm_action`, `hudl_vm_attach_network` |
| **Volumes** | `hudl_volume_list`, `hudl_volume_get`, `hudl_volume_create`, `hudl_volume_delete`, `hudl_volume_attach`, `hudl_volume_detach` |
| **Floating IPs** | `hudl_fip_list`, `hudl_fip_get`, `hudl_fip_associate`, `hudl_fip_disassociate` |
| **Security Groups** | `hudl_sg_list`, `hudl_sg_get`, `hudl_sg_create`, `hudl_sg_delete`, `hudl_sg_duplicate`, `hudl_sg_rule_add`, `hudl_sg_rule_delete` |
| **Networks** | `hudl_network_list`, `hudl_network_create`, `hudl_network_delete` |
| **SSH Keys** | `hudl_key_list`, `hudl_key_get`, `hudl_key_create`, `hudl_key_delete` |
| **Lookup** | `hudl_flavor_list`, `hudl_image_list`, `hudl_region_list` |
| **GPU Marketplace** | `hudl_gpu_offers`, `hudl_gpu_summary`, `hudl_gpu_check` |
| **GPU Deployments** | `hudl_gpu_list`, `hudl_gpu_get`, `hudl_gpu_deploy`, `hudl_gpu_action`, `hudl_gpu_delete` |
| **GPU Waitlist** | `hudl_gpu_waitlist_list`, `hudl_gpu_waitlist_add`, `hudl_gpu_waitlist_cancel` |
| **GPU Images** | `hudl_gpu_image_list` |
| **GPU Volumes** | `hudl_gpu_volume_list`, `hudl_gpu_volume_create`, `hudl_gpu_volume_delete` |
| **GPU SSH Keys** | `hudl_gpu_ssh_key_list`, `hudl_gpu_ssh_key_upload`, `hudl_gpu_ssh_key_delete` |
| **GPU API Keys** | `hudl_gpu_api_key_list`, `hudl_gpu_api_key_create`, `hudl_gpu_api_key_revoke` |
| **GPU Webhooks** | `hudl_gpu_webhook_list`, `hudl_gpu_webhook_create`, `hudl_gpu_webhook_update`, `hudl_gpu_webhook_delete` |
| **GPU Regions** | `hudl_gpu_region_list`, `hudl_gpu_volume_type_list` |

## Configuration

Stored in `~/.hudl/config.toml`:

```toml
api_key = "your-cloud-api-key"
gpu_api_key = "your-gpu-api-key"
workspace = "my-org"
region = "eu2"
```

Override with flags (`--api-key`, `--gpu-api-key`, `--workspace`, `--region`) or environment variables (`HUDL_API_KEY`, `HUDL_GPU_API_KEY`, `HUDL_WORKSPACE`, `HUDL_REGION`).

## Development

**Prerequisites:** Go 1.25+, `make`

```sh
make build          # Build CLI binary
make build-mcp      # Build MCP server binary
make build-all      # Build both CLI and MCP server
make dev            # Build and run the CLI
make test           # Run tests
make dist           # Cross-compile all platforms (CLI + MCP)
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
├── internal/
│   ├── config/                # Config file management (shared)
│   └── runtime/               # HTTP client, I/O, app context (shared)
├── cli/
│   ├── cmd/hudl/              # CLI entrypoint
│   └── internal/cli/          # Cobra commands
├── mcp/
│   ├── cmd/hudl-mcp/          # MCP server entrypoint
│   └── internal/
│       ├── server/            # JSON-RPC / MCP protocol engine
│       └── tools/             # Tool definitions & handlers
├── get/                       # get.huddle01.com
│   └── static/
│       ├── index.html         # Landing page
│       └── install.sh         # Shell installer
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
