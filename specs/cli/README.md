# Huddle01 CLI — Specification

The `hudl` binary is the official command-line interface for managing Huddle01 Cloud and GPU infrastructure. It provides a consistent, scriptable interface with human-friendly TTY output, JSON/YAML machine output, and interactive prompts.

---

## Table of Contents

- [Architecture](#architecture)
- [Installation & Build](#installation--build)
- [Configuration](#configuration)
- [Authentication](#authentication)
- [Output Modes](#output-modes)
- [Command Reference](#command-reference)
  - [Auth & Context](#auth--context)
  - [Cloud Infrastructure](#cloud-infrastructure)
  - [GPU Marketplace & Deployments](#gpu-marketplace--deployments)
- [Request Loading & Idempotency](#request-loading--idempotency)
- [Error Handling](#error-handling)
- [Internals](#internals)
- [Contributing](#contributing)

---

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                       User / Script                       │
└─────────────────────────┬────────────────────────────────┘
                          │  CLI invocation
                          ▼
┌──────────────────────────────────────────────────────────┐
│               cli/internal/cli  (Cobra commands)         │
│  ┌──────────┐ ┌────────┐ ┌───────┐ ┌─────┐ ┌────────┐  │
│  │ auth.go  │ │cloud.go│ │ gpu.go│ │root │ │helpers │  │
│  └──────────┘ └────────┘ └───────┘ └─────┘ └────────┘  │
└─────────────────────────┬────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        ▼                 ▼                 ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────────┐
│internal/config│ │internal/     │ │internal/runtime/ │
│  config.go   │ │runtime/app.go│ │ http.go          │
│              │ │              │ │ output.go        │
│  TOML load   │ │  App context │ │ input.go         │
│  Env merge   │ │  TTY detect  │ │ HTTP client      │
│  Flag merge  │ │  I/O writers │ │ Retry + backoff  │
└──────────────┘ └──────────────┘ └──────────────────┘
                          │
          ┌───────────────┼───────────────┐
          ▼                               ▼
┌──────────────────┐            ┌──────────────────┐
│  Cloud API       │            │  GPU API         │
│  cloud.huddleapis│            │  gpu.huddleapis  │
│  .com/api/v1     │            │  .com/api/v1     │
└──────────────────┘            └──────────────────┘
```

The CLI and MCP server share the `internal/config` and `internal/runtime` packages. This means config resolution, HTTP transport, retry logic, and error types are identical across both surfaces.

---

## Installation & Build

### Prerequisites

- Go 1.23 or later
- Make (optional, for convenience targets)

### Build

```bash
# Build the CLI binary
make build          # produces ./hudl

# Build both CLI and MCP binaries
make build-all      # produces ./hudl and ./hudl-mcp

# Cross-compile release binaries
make dist           # linux/amd64 and darwin/arm64 for both binaries
```

### Install from source

```bash
go install ./cli/cmd/hudl@latest
```

---

## Configuration

Configuration is resolved in a layered cascade. Each layer overrides the previous:

```
User config  →  Project config  →  Environment variables  →  CLI flags
(~/.hudl/config.toml)  (./hudl.toml)
```

### User Config (`~/.hudl/config.toml`)

```toml
api_key   = "hk_abc123..."
workspace = "my-team"
region    = "eu2"
output    = "table"

[api]
cloud_base_url = "https://cloud.huddleapis.com/api/v1"   # default
gpu_base_url   = "https://gpu.huddleapis.com/api/v1"     # default

[defaults]
# Command-specific defaults (reserved for future use)
```

### Project Config (`./hudl.toml`)

Same structure as the user config. Place in your project root to share settings across a team.

### Environment Variables

| Variable              | Overrides         |
|-----------------------|-------------------|
| `HUDL_API_KEY`        | `api_key`         |
| `HUDL_WORKSPACE`      | `workspace`       |
| `HUDL_REGION`         | `region`          |
| `HUDL_OUTPUT`         | `output`          |
| `HUDL_CLOUD_BASE_URL` | `api.cloud_base_url` |
| `HUDL_GPU_BASE_URL`   | `api.gpu_base_url`   |

### CLI Flags (highest precedence)

```bash
hudl --api-key <key> --workspace <ws> --region <r> --output json vm list
```

### Global Flags

| Flag                  | Short | Description                           |
|-----------------------|-------|---------------------------------------|
| `--api-key`           |       | API key for authentication            |
| `--workspace`         | `-w`  | Active workspace                      |
| `--region`            | `-r`  | Target region                         |
| `--output`            | `-o`  | Output format (table, json, yaml, wide, name) |
| `--timeout`           |       | HTTP request timeout                  |
| `--verbose`           | `-v`  | Enable verbose/debug output           |
| `--no-color`          |       | Disable colored output                |
| `--quiet`             | `-q`  | Suppress non-essential output         |

---

## Authentication

### Storing a key

```bash
# Interactive login
hudl login --token hk_abc123...

# Or set via environment
export HUDL_API_KEY=hk_abc123...
```

### Checking auth status

```bash
hudl auth status
```

Output (TTY):

```
API Key:    hk_a****************************23ef
Workspace:  my-team
Region:     eu2
Config:     /Users/you/.hudl/config.toml
```

### Clearing auth

```bash
hudl auth clear
```

### How auth is used

The CLI sends the API key differently depending on the backend:

| Backend | Header                           |
|---------|----------------------------------|
| Cloud   | `X-API-Key: <key>`               |
| GPU     | `Authorization: Bearer <key>`    |

---

## Output Modes

The `--output` / `-o` flag controls how results are rendered.

| Mode    | Description                                            |
|---------|--------------------------------------------------------|
| `table` | Human-readable tab-aligned columns (default for TTY)   |
| `json`  | Pretty-printed JSON with 2-space indent                |
| `yaml`  | YAML output                                            |
| `wide`  | Table with all available fields                        |
| `name`  | One ID/name per line (useful for piping)               |

### Smart table layout

When using `table` or `wide` mode, the CLI:

1. Prioritizes common columns: `id`, `name`, `status`, `region`, timestamps
2. Flattens nested objects (e.g., `region.name` extracted from `{region: {name: "eu2"}}`)
3. Truncates long values for readability
4. Groups related columns together

### Piping and scripts

```bash
# Get all VM IDs
hudl vm list -o name

# JSON output for jq processing
hudl vm get abc-123 -o json | jq '.status'

# YAML for config generation
hudl vm get abc-123 -o yaml
```

---

## Command Reference

### Auth & Context

| Command                    | Description                          |
|----------------------------|--------------------------------------|
| `hudl login --token <key>` | Store API key                        |
| `hudl auth status`         | Show auth state                      |
| `hudl auth clear`          | Remove stored API key                |
| `hudl ctx use <workspace>` | Set default workspace                |
| `hudl ctx region <region>` | Set default region                   |

### Cloud Infrastructure

All cloud commands require `--region` (or a default region set via `ctx region`).

#### Virtual Machines

| Command                            | Description                       |
|------------------------------------|-----------------------------------|
| `hudl vm list`                     | List all VMs in region            |
| `hudl vm get <id>`                 | Get VM details                    |
| `hudl vm create <name>`            | Create a new VM                   |
| `hudl vm delete <id>`              | Delete a VM (with confirmation)   |
| `hudl vm status <id>`              | Get VM power/lifecycle status     |
| `hudl vm action <id> <action>`     | Run lifecycle action (e.g. reboot)|
| `hudl vm attach-network <id>`      | Attach a network to a VM          |

**VM create flags:**

| Flag              | Required | Description                           |
|-------------------|----------|---------------------------------------|
| `--flavor`        | Yes      | Compute flavor ID                     |
| `--image`         | Yes      | Image ID for boot disk                |
| `--boot-disk-size`|          | Boot disk size in GB                  |
| `--key`           |          | SSH key pair name (repeatable)        |
| `--sg`            |          | Security group ID (repeatable)        |
| `--file`          |          | Load request body from JSON/YAML file |

#### Volumes

| Command                   | Description                        |
|---------------------------|------------------------------------|
| `hudl volume list`        | List all volumes in region         |
| `hudl volume get <id>`    | Get volume details                 |
| `hudl volume create`      | Create a new volume                |
| `hudl volume delete <id>` | Delete a volume                    |
| `hudl volume attach <id>` | Attach volume to an instance       |
| `hudl volume detach <id>` | Detach volume from an instance     |

#### Floating IPs

| Command                               | Description                   |
|---------------------------------------|-------------------------------|
| `hudl floating-ip list`               | List floating IPs             |
| `hudl floating-ip get <id>`           | Get floating IP details       |
| `hudl floating-ip associate <id>`     | Associate with an instance    |
| `hudl floating-ip disassociate <id>`  | Remove association            |

#### Security Groups

| Command                                        | Description                      |
|------------------------------------------------|----------------------------------|
| `hudl sg list`                                 | List security groups             |
| `hudl sg get <id>`                             | Get security group details       |
| `hudl sg create <name>`                        | Create security group            |
| `hudl sg delete <id>`                          | Delete security group            |
| `hudl sg duplicate <id>`                       | Copy SG to another region        |
| `hudl sg rule add <sg-id>`                     | Add an ingress/egress rule       |
| `hudl sg rule delete <sg-id> <rule-id>`        | Remove a rule                    |

**SG rule add flags:**

| Flag            | Description                                  |
|-----------------|----------------------------------------------|
| `--direction`   | `ingress` or `egress`                        |
| `--protocol`    | `tcp`, `udp`, `icmp`                         |
| `--port-min`    | Start of port range                          |
| `--port-max`    | End of port range                            |
| `--remote-cidr` | Source/destination CIDR (e.g. `0.0.0.0/0`)   |

#### Networks

| Command                     | Description                  |
|-----------------------------|------------------------------|
| `hudl network list`         | List networks in region      |
| `hudl network create <name>`| Create a network             |
| `hudl network delete <id>`  | Delete a network             |

**Network create flags:**

| Flag          | Description                    |
|---------------|--------------------------------|
| `--cidr`      | Subnet CIDR                    |
| `--gateway`   | Gateway IP address             |
| `--dhcp`      | Enable DHCP (true/false)       |

#### Key Pairs

| Command                   | Description              |
|---------------------------|--------------------------|
| `hudl key list`           | List SSH key pairs       |
| `hudl key get <name>`     | Get key pair details     |
| `hudl key create <name>`  | Create/import key pair   |
| `hudl key delete <name>`  | Delete key pair          |

#### Discovery

| Command            | Description                            |
|--------------------|----------------------------------------|
| `hudl flavor list` | List available compute flavors         |
| `hudl image list`  | List available OS images (by distro)   |
| `hudl region list` | List available cloud regions           |

### GPU Marketplace & Deployments

GPU commands use the GPU API backend (`gpu.huddleapis.com`).

#### Marketplace

| Command                           | Description                        |
|-----------------------------------|------------------------------------|
| `hudl gpu offers`                 | List available GPU offers          |
| `hudl gpu summary`                | GPU marketplace summary            |
| `hudl gpu check <cluster-type>`   | Check availability for a type      |

**GPU offers flags:**

| Flag            | Description                                |
|-----------------|--------------------------------------------|
| `--cluster-type`| Filter by cluster type                     |
| `--region`      | Filter by region                           |
| `--min-gpu`     | Minimum GPU count                          |
| `--max-gpu`     | Maximum GPU count                          |
| `--sort`        | Sort field                                 |
| `--order`       | Sort order (asc/desc)                      |
| `--page`        | Page number                                |
| `--page-size`   | Results per page                           |

#### Deployments

| Command                             | Description                  |
|-------------------------------------|------------------------------|
| `hudl gpu list`                     | List GPU deployments         |
| `hudl gpu get <id>`                 | Get deployment details       |
| `hudl gpu deploy`                   | Deploy a new GPU cluster     |
| `hudl gpu action <id> <action>`     | Run action on deployment     |
| `hudl gpu delete <id>`              | Delete a deployment          |

**GPU deploy flags:**

| Flag              | Description                    |
|-------------------|--------------------------------|
| `--cluster-type`  | GPU cluster type               |
| `--region`        | Target region                  |
| `--gpu-count`     | Number of GPUs                 |
| `--image`         | OS image ID                    |
| `--ssh-key`       | SSH key name                   |
| `--volume`        | Volume ID to attach            |
| `--file`          | Load request from JSON/YAML    |

#### Waitlist

| Command                           | Description                      |
|-----------------------------------|----------------------------------|
| `hudl gpu waitlist list`          | List waitlist requests           |
| `hudl gpu waitlist add`           | Create a waitlist request        |
| `hudl gpu waitlist cancel <id>`   | Cancel a waitlist request        |

#### GPU Resources

| Command                                | Description                     |
|----------------------------------------|---------------------------------|
| `hudl gpu image list`                  | List GPU images                 |
| `hudl gpu volume list`                 | List GPU volumes                |
| `hudl gpu volume create`               | Create a GPU volume             |
| `hudl gpu key list`                    | List SSH keys                   |
| `hudl gpu key upload`                  | Upload SSH key                  |
| `hudl gpu key delete <name>`           | Delete SSH key                  |
| `hudl gpu apikey list`                 | List API keys                   |
| `hudl gpu apikey create`               | Create API key                  |
| `hudl gpu apikey revoke <id>`          | Revoke API key                  |
| `hudl gpu webhook list`               | List webhooks                   |
| `hudl gpu webhook create`             | Create webhook                  |
| `hudl gpu webhook update <id>`        | Update webhook                  |
| `hudl gpu webhook delete <id>`        | Delete webhook                  |
| `hudl gpu region list`                | List GPU regions                |
| `hudl gpu region volume-types`        | List available volume types     |

---

## Request Loading & Idempotency

### Loading requests from files

For complex create operations, you can load the request body from a file:

```bash
# From JSON file
hudl vm create my-vm --file request.json

# From YAML file
hudl vm create my-vm --file request.yaml

# From stdin
cat request.json | hudl vm create my-vm --file -
```

File flags override fields in the loaded file, so you can use a template and selectively override:

```bash
hudl vm create my-vm --file base.json --flavor custom-8vcpu
```

### Dry run

Preview what would be sent without making the API call:

```bash
hudl vm create my-vm --flavor f1 --image img1 --dry-run
```

### Idempotency

All mutating requests (create, delete, actions) automatically include an idempotency key (`Idempotency-Key` header) generated as a UUID with the `hudl_` prefix. You can also supply your own:

```bash
hudl vm create my-vm --idempotency-key my-unique-key-123
```

This ensures that retried requests due to network issues do not create duplicate resources.

---

## Error Handling

### HTTP errors

When an API call fails, the CLI renders a structured error:

**TTY mode (table output):**
```
Error: 404 Not Found — instance abc-123 not found
Request ID: req_xyz789
```

**JSON output mode:**
```json
{
  "error": {
    "status_code": 404,
    "message": "instance abc-123 not found",
    "request_id": "req_xyz789"
  }
}
```

### Retry behavior

The HTTP client automatically retries on transient failures:

| Condition              | Behavior                                         |
|------------------------|--------------------------------------------------|
| HTTP 429 (Rate Limit)  | Retry with exponential backoff                   |
| HTTP 5xx (Server Error) | Retry with exponential backoff                  |
| Network/DNS errors     | Retry with exponential backoff                   |
| HTTP 4xx (Client Error) | Fail immediately (no retry)                     |

Retry timing: up to 4 attempts with jitter-based exponential backoff (`base^attempt × 150ms + random(0–100ms)`).

### Confirmation prompts

Destructive operations (delete, clear) prompt for confirmation in interactive TTY sessions:

```bash
$ hudl vm delete abc-123
Delete instance abc-123? [y/N]: y
```

Skip with `--yes` or `-y`:

```bash
hudl vm delete abc-123 --yes
```

Non-TTY sessions (pipes, scripts) skip prompts automatically.

---

## Internals

### Package structure

```
cli/
├── cmd/hudl/
│   └── main.go              # Entry point, version embedding
└── internal/cli/
    ├── root.go              # Root command, global flags, subcommand registration
    ├── auth.go              # login, auth, ctx commands
    ├── cloud.go             # VM, volume, floating-ip, sg, network, key, flavor, image, region
    ├── gpu.go               # GPU offers, deployments, waitlist, images, volumes, keys, webhooks
    └── helpers.go           # Output formatting, request handling, input validation

internal/
├── config/
│   └── config.go            # TOML loading, env/flag merge, config resolution
└── runtime/
    ├── app.go               # App context, global options, TTY detection
    ├── http.go              # HTTP client, retry, backends, HTTPError
    ├── output.go            # JSON/YAML/table/wide/name rendering
    └── input.go             # Interactive prompts, request file loading
```

### Command registration pattern

Every command group follows a consistent builder pattern:

```go
func newVMCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "vm",
        Short: "Manage virtual machines",
    }
    cmd.AddCommand(
        newVMListCommand(),
        newVMGetCommand(),
        newVMCreateCommand(),
        // ...
    )
    return cmd
}
```

Each leaf command:

1. Extracts the `App` from Cobra's context via `appFromCommand(cmd)`
2. Builds a request map from flags and/or `--file`
3. Calls `handleRequest()` which sends the HTTP request (or prints dry-run output)
4. Formats and renders the response via `executeResult()`

### Middleware stack

The root command's `PersistentPreRunE` runs before every subcommand:

1. Reads global flags
2. Loads config via `config.Load()` with the flag overrides
3. Creates the `runtime.App` (HTTP client, TTY detection, I/O writers)
4. Injects `App` into the Cobra command context

### Shared with MCP

The `internal/config` and `internal/runtime` packages are shared between the CLI and MCP server. This guarantees:

- Identical config resolution (same TOML parsing, env var support, defaults)
- Identical HTTP behavior (same retry logic, backoff, headers, error types)
- Identical API compatibility (same base URLs, path construction, auth headers)

Changes to the shared packages affect both the CLI and MCP simultaneously.

---

## Contributing

### Adding a new command

1. Add the command function in the appropriate file (`cloud.go`, `gpu.go`, or a new file)
2. Follow the existing pattern: `newXxxCommand()` returns `*cobra.Command`
3. Register it in `root.go` via `rootCmd.AddCommand()`
4. Use `handleRequest()` for API calls to get automatic dry-run, output formatting, and error handling
5. Add shell completion hints where appropriate using `completeCloudResource()`

### Adding a new global flag

1. Add the field to `runtime.GlobalOptions` in `internal/runtime/app.go`
2. Add the env var mapping in `internal/config/config.go`
3. Bind the flag in `root.go`'s root command setup

### Testing

```bash
# Run all tests
make test

# Vet the code
make vet

# Build to verify compilation
make build
```
