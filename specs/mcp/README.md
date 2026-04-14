# Huddle01 MCP Server — Specification

The `hudl-mcp` binary is a production-grade [Model Context Protocol](https://modelcontextprotocol.io) server that exposes the full Huddle01 Cloud API as 67 structured tools. It is designed for AI agents, IDE assistants, and automation workflows.

---

## Table of Contents

- [Architecture](#architecture)
- [Protocol](#protocol)
- [Installation & Build](#installation--build)
- [Configuration](#configuration)
- [Running the Server](#running-the-server)
- [Client Integration](#client-integration)
- [Tool Reference](#tool-reference)
- [Error Handling](#error-handling)
- [Security](#security)
- [Internals](#internals)
- [Contributing](#contributing)

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│  MCP Client (Claude, Cursor, IDE, Agent, etc.)  │
└──────────────────────┬──────────────────────────┘
                       │ JSON-RPC 2.0 over stdio
                       ▼
┌──────────────────────────────────────────────────┐
│  hudl-mcp                                        │
│                                                  │
│  ┌────────────────────────────────────────────┐  │
│  │  server/server.go                          │  │
│  │  JSON-RPC router, MCP protocol handler     │  │
│  └────────────────┬───────────────────────────┘  │
│                   │                              │
│  ┌────────────────▼───────────────────────────┐  │
│  │  tools/*.go                                │  │
│  │  67 tool handlers (auth, cloud, GPU, etc.) │  │
│  └────────────────┬───────────────────────────┘  │
│                   │                              │
│  ┌────────────────▼───────────────────────────┐  │
│  │  internal/runtime (shared with CLI)        │  │
│  │  HTTP client, retry logic, config loader   │  │
│  └────────────────┬───────────────────────────┘  │
└───────────────────┼──────────────────────────────┘
                    │ HTTPS
                    ▼
        ┌───────────────────────┐
        │  Huddle01 Cloud APIs  │
        │  cloud.huddleapis.com │
        │  gpu.huddleapis.com   │
        └───────────────────────┘
```

### Key Design Decisions

1. **Shared core** — The MCP server reuses the same `internal/config` and `internal/runtime` packages as the CLI. Same HTTP client, same retry logic, same config resolution.

2. **Zero external dependencies** — The MCP protocol layer is pure Go. No MCP framework, no JSON-RPC library. This keeps the binary small (~6MB) and avoids dependency churn.

3. **Stdio transport** — The server communicates over stdin/stdout using newline-delimited JSON-RPC. This is the standard MCP transport and works with every client.

4. **Cached client** — Config is loaded from disk once and cached. It is only re-read after auth or context mutations (`hudl_login`, `hudl_auth_clear`, `hudl_ctx_use`, `hudl_ctx_region`).

5. **Structured errors** — HTTP errors from the Huddle01 API preserve their status code, message, request ID, and body in the error text as JSON, so AI agents can programmatically distinguish a 401 from a 404.

---

## Protocol

The server implements MCP protocol version `2024-11-05` over JSON-RPC 2.0.

### Supported Methods

| Method | Direction | Description |
|--------|-----------|-------------|
| `initialize` | Client → Server | Exchange capabilities and protocol version |
| `notifications/initialized` | Client → Server | Client acknowledges init (no response) |
| `tools/list` | Client → Server | Returns all 67 tool definitions with JSON Schema |
| `tools/call` | Client → Server | Execute a tool by name with arguments |
| `ping` | Client → Server | Health check, returns `{}` |

### Message Format

Every message is a single line of JSON (newline-delimited):

**Request:**
```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hudl_vm_list","arguments":{}}}
```

**Response (success):**
```json
{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"[{\"id\":\"vm-123\",...}]"}]}}
```

**Response (error):**
```json
{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{\"status_code\":401,\"message\":\"unauthorized\"}"}],"isError":true}}
```

### Initialization Handshake

```
Client → {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"my-client","version":"1.0"}}}
Server → {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"hudl-mcp","version":"..."}}}
Client → {"jsonrpc":"2.0","method":"notifications/initialized"}
```

---

## Installation & Build

### From Source

```sh
# Build MCP server only
make build-mcp

# Build both CLI and MCP server
make build-all

# Cross-compile for all platforms
make dist
```

### Binary Location

After `make build-mcp`, the binary is at `./hudl-mcp` in the repo root.

After `make dist`, platform binaries are at `dist/hudl-mcp-{os}-{arch}`.

### Version

```sh
./hudl-mcp --version
./hudl-mcp --help
```

---

## Configuration

The MCP server reads the same configuration as the CLI, in this priority order:

1. **User config** — `~/.hudl/config.toml`
2. **Project config** — `./hudl.toml` (in working directory)
3. **Environment variables** — `HUDL_API_KEY`, `HUDL_REGION`, `HUDL_WORKSPACE`, etc.

### Required Configuration

At minimum, you need an API key for each backend you want to use:

```sh
# Option 1: Set via CLI
hudl login --token sk_your_cloud_key --gpu-token sk_your_gpu_key

# Option 2: Environment variables
export HUDL_API_KEY=sk_your_cloud_key
export HUDL_GPU_API_KEY=sk_your_gpu_key

# Option 3: Use the MCP tool itself
# (send hudl_login tool call with token and/or gpu_token arguments)
```

If only `HUDL_API_KEY` is set, it will be used as a fallback for GPU requests too.

### Config File Format

```toml
api_key = "sk_your_cloud_api_key"
gpu_api_key = "sk_your_gpu_api_key"
workspace = "my-org"
region = "eu2"

[api]
cloud_base_url = "https://cloud.huddleapis.com/api/v1"
gpu_base_url = "https://gpu.huddleapis.com/api/v1"
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `HUDL_API_KEY` | Cloud API key for authentication |
| `HUDL_GPU_API_KEY` | GPU API key for authentication (falls back to `HUDL_API_KEY`) |
| `HUDL_WORKSPACE` | Default workspace |
| `HUDL_REGION` | Default region |
| `HUDL_OUTPUT` | Output format (not used by MCP, always JSON) |
| `HUDL_CLOUD_BASE_URL` | Override cloud API base URL |
| `HUDL_GPU_BASE_URL` | Override GPU API base URL |

---

## Running the Server

The server reads from stdin and writes to stdout. It is launched by the MCP client.

```sh
# Direct (for testing)
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./hudl-mcp

# List all tools
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | ./hudl-mcp 2>/dev/null
```

Diagnostic messages are written to stderr and never interfere with the JSON-RPC protocol on stdout.

---

## Client Integration

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "huddle01": {
      "command": "/absolute/path/to/hudl-mcp"
    }
  }
}
```

### Claude Code

Add to `.claude/settings.json` or `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "huddle01": {
      "command": "/absolute/path/to/hudl-mcp"
    }
  }
}
```

### Cursor

Add to Cursor's MCP settings:

```json
{
  "mcpServers": {
    "huddle01": {
      "command": "/absolute/path/to/hudl-mcp"
    }
  }
}
```

### With Environment Variables

If you prefer to pass the API key via environment rather than config file:

```json
{
  "mcpServers": {
    "huddle01": {
      "command": "/absolute/path/to/hudl-mcp",
      "env": {
        "HUDL_API_KEY": "sk_your_cloud_key",
        "HUDL_GPU_API_KEY": "sk_your_gpu_key",
        "HUDL_REGION": "eu2"
      }
    }
  }
}
```

### Programmatic Usage

Any process can launch `hudl-mcp` and communicate over stdin/stdout:

```python
import subprocess, json

proc = subprocess.Popen(["./hudl-mcp"], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

def call(method, params=None, id=1):
    msg = {"jsonrpc": "2.0", "id": id, "method": method}
    if params:
        msg["params"] = params
    proc.stdin.write(json.dumps(msg).encode() + b"\n")
    proc.stdin.flush()
    return json.loads(proc.stdout.readline())

# Initialize
call("initialize", {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "test", "version": "1.0"}})
proc.stdin.write(b'{"jsonrpc":"2.0","method":"notifications/initialized"}\n')
proc.stdin.flush()

# List VMs
result = call("tools/call", {"name": "hudl_vm_list", "arguments": {}}, id=2)
print(result)
```

---

## Tool Reference

### Naming Convention

All tools use the prefix `hudl_` followed by the resource and action:

```
hudl_{resource}_{action}
hudl_{resource}_{subresource}_{action}
```

Examples: `hudl_vm_create`, `hudl_sg_rule_add`, `hudl_gpu_ssh_key_upload`

### Authentication & Context (6 tools)

| Tool | Description |
|------|-------------|
| `hudl_login` | Store API keys. Optional: `token` (Cloud), `gpu_token` (GPU) |
| `hudl_auth_status` | Show auth state (Cloud & GPU keys, workspace, region) |
| `hudl_auth_clear` | Remove saved API keys (both Cloud and GPU) |
| `hudl_ctx_show` | Show current workspace/region |
| `hudl_ctx_use` | Set default workspace. Required: `workspace` |
| `hudl_ctx_region` | Set default region. Required: `region` |

### Virtual Machines (7 tools)

| Tool | Description | Required Args |
|------|-------------|---------------|
| `hudl_vm_list` | List VMs in current region | — |
| `hudl_vm_get` | Get VM details | `id` |
| `hudl_vm_create` | Create a VM | `name`, `flavor_id`, `image_id`, `boot_disk_size`, `key_name`, `sg_names` |
| `hudl_vm_delete` | Delete a VM | `id` |
| `hudl_vm_status` | Get VM status | `id` |
| `hudl_vm_action` | Lifecycle action (start/stop/reboot) | `id`, `action` |
| `hudl_vm_attach_network` | Attach network to VM | `id`, `network_id` |

### Block Storage (6 tools)

| Tool | Description | Required Args |
|------|-------------|---------------|
| `hudl_volume_list` | List volumes | — |
| `hudl_volume_get` | Get volume details | `id` |
| `hudl_volume_create` | Create volume | `name`, `size` |
| `hudl_volume_delete` | Delete volume | `id` |
| `hudl_volume_attach` | Attach to VM | `id`, `instance_id` |
| `hudl_volume_detach` | Detach from VM | `id`, `instance_id` |

### Floating IPs (4 tools)

| Tool | Description | Required Args |
|------|-------------|---------------|
| `hudl_fip_list` | List floating IPs | — |
| `hudl_fip_get` | Get floating IP details | `id` |
| `hudl_fip_associate` | Associate with VM | `id`, `instance_id` |
| `hudl_fip_disassociate` | Disassociate from VM | `id` |

### Security Groups (8 tools)

| Tool | Description | Required Args |
|------|-------------|---------------|
| `hudl_sg_list` | List security groups | — |
| `hudl_sg_get` | Get security group with rules | `id` |
| `hudl_sg_create` | Create security group | `name` |
| `hudl_sg_delete` | Delete security group | `id` |
| `hudl_sg_duplicate` | Duplicate to another region | `id`, `target_region` |
| `hudl_sg_rule_add` | Add firewall rule | `sg_id`, `direction`, `ether_type` |
| `hudl_sg_rule_delete` | Delete firewall rule | `sg_id`, `rule_id` |

### Networks (3 tools)

| Tool | Description | Required Args |
|------|-------------|---------------|
| `hudl_network_list` | List networks | — |
| `hudl_network_create` | Create network | `name` |
| `hudl_network_delete` | Delete network | `id` |

### SSH Keys (4 tools)

| Tool | Description | Required Args |
|------|-------------|---------------|
| `hudl_key_list` | List key pairs | — |
| `hudl_key_get` | Get key pair | `name` |
| `hudl_key_create` | Create key pair | `name`, `public_key` |
| `hudl_key_delete` | Delete key pair | `name` |

### Lookup (3 tools)

| Tool | Description |
|------|-------------|
| `hudl_flavor_list` | List compute flavors (for `vm_create`) |
| `hudl_image_list` | List OS images (for `vm_create`) |
| `hudl_region_list` | List available regions |

### GPU Marketplace (3 tools)

| Tool | Description | Required Args |
|------|-------------|---------------|
| `hudl_gpu_offers` | List GPU marketplace offers | — (all optional filters) |
| `hudl_gpu_summary` | GPU marketplace summary | — |
| `hudl_gpu_check` | Check cluster type availability | `cluster_type` |

### GPU Deployments (5 tools)

| Tool | Description | Required Args |
|------|-------------|---------------|
| `hudl_gpu_list` | List deployments | — |
| `hudl_gpu_get` | Get deployment details | `id` |
| `hudl_gpu_deploy` | Deploy GPU cluster | `cluster_type`, `image`, `hostname`, `location`, `ssh_key_ids` |
| `hudl_gpu_action` | Run action on deployment | `id`, `action` |
| `hudl_gpu_delete` | Delete deployment | `id` |

### GPU Waitlist (3 tools)

| Tool | Description | Required Args |
|------|-------------|---------------|
| `hudl_gpu_waitlist_list` | List waitlist requests | — |
| `hudl_gpu_waitlist_add` | Create waitlist request | `cluster_type` |
| `hudl_gpu_waitlist_cancel` | Cancel waitlist request | `id` |

### GPU Images (1 tool)

| Tool | Description |
|------|-------------|
| `hudl_gpu_image_list` | List GPU images (optional: `cluster_type`, `image_type`) |

### GPU Volumes (3 tools)

| Tool | Description | Required Args |
|------|-------------|---------------|
| `hudl_gpu_volume_list` | List GPU volumes | — |
| `hudl_gpu_volume_create` | Create GPU volume | `name`, `type`, `location`, `size` |
| `hudl_gpu_volume_delete` | Delete GPU volume | `id` |

### GPU SSH Keys (3 tools)

| Tool | Description | Required Args |
|------|-------------|---------------|
| `hudl_gpu_ssh_key_list` | List GPU SSH keys | — |
| `hudl_gpu_ssh_key_upload` | Upload GPU SSH key | `name`, `public_key` |
| `hudl_gpu_ssh_key_delete` | Delete GPU SSH key | `id` |

### GPU API Keys (3 tools)

| Tool | Description | Required Args |
|------|-------------|---------------|
| `hudl_gpu_api_key_list` | List GPU API keys | — |
| `hudl_gpu_api_key_create` | Create GPU API key | `name` |
| `hudl_gpu_api_key_revoke` | Revoke GPU API key | `id` |

### GPU Webhooks (4 tools)

| Tool | Description | Required Args |
|------|-------------|---------------|
| `hudl_gpu_webhook_list` | List webhooks | — |
| `hudl_gpu_webhook_create` | Create webhook | `url`, `events` |
| `hudl_gpu_webhook_update` | Update webhook | `id` |
| `hudl_gpu_webhook_delete` | Delete webhook | `id` |

### GPU Regions (2 tools)

| Tool | Description |
|------|-------------|
| `hudl_gpu_region_list` | List GPU regions/locations |
| `hudl_gpu_volume_type_list` | List GPU volume types |

---

## Error Handling

### Error Format

Tool errors are returned as `isError: true` in the MCP result. For HTTP errors from the Huddle01 API, the error text is a JSON object with structured fields:

```json
{
  "status_code": 401,
  "message": "Invalid API key",
  "request_id": "req_abc123",
  "body": { "error": "unauthorized" }
}
```

### Common Error Codes

| Status | Meaning | Resolution |
|--------|---------|------------|
| 401 | Unauthorized | Run `hudl_login` with a valid API key |
| 403 | Forbidden | Check workspace permissions |
| 404 | Not found | Verify the resource ID exists |
| 422 | Validation error | Check required fields in the request body |
| 429 | Rate limited | Automatic retry with exponential backoff (up to 4 attempts) |
| 5xx | Server error | Automatic retry with exponential backoff (up to 4 attempts) |

### Config Errors

If config can't be loaded or a required field (like `region`) is missing, the error is a plain string:

```
"region is required; set HUDL_REGION or run `hudl ctx region <region>`"
```

---

## Security

### API Key Storage

- API keys (Cloud and GPU) are stored in `~/.hudl/config.toml` with file permissions `0600`
- Keys are never logged or written to stdout
- The `hudl_auth_status` tool masks both keys (shows first/last 4 characters)

### Transport Security

- All API calls use HTTPS to `cloud.huddleapis.com` and `gpu.huddleapis.com`
- The stdio transport between client and server is local (no network)

### Mutating Operations

- All write operations (create, delete, action) include an auto-generated idempotency key
- This prevents duplicate operations if a tool call is retried

---

## Internals

### Source Structure

```
mcp/
├── cmd/hudl-mcp/
│   └── main.go              # Entry point (25 lines)
└── internal/
    ├── server/
    │   └── server.go         # MCP protocol engine (~290 lines)
    └── tools/
        ├── tools.go          # Shared helpers, config caching, error handling
        ├── auth.go           # Auth & context tools (6 tools)
        ├── cloud.go          # Cloud compute tools (33 tools)
        └── gpu.go            # GPU tools (28 tools)
```

### Shared Packages (with CLI)

```
internal/
├── config/config.go          # Config loading, TOML parsing, env/flag merging
└── runtime/
    ├── app.go                # App context, global options
    ├── http.go               # HTTP client with retry, error types
    ├── input.go              # User prompts (CLI only), request loading
    └── output.go             # Output formatting (CLI only)
```

### Config Caching

The MCP server loads `~/.hudl/config.toml` once at first use and caches the `runtime.App` instance. This means:

- No disk I/O per tool call
- TCP connections to the API are reused via `http.Client`
- The cache is invalidated when auth or context is mutated via MCP tools

### Retry Logic

Inherited from the shared HTTP client:

- **Retries on:** HTTP 429, 5xx, network errors
- **Attempts:** Up to 4
- **Backoff:** Exponential with jitter: `2^attempt × 150ms + random(0-100ms)`

---

## Contributing

### Adding a New Tool

1. Choose the right file: `auth.go`, `cloud.go`, or `gpu.go`
2. Define the tool with `server.Tool{Name, Description, InputSchema}`
3. Implement the handler function
4. Register it in the appropriate `register*Tools` function
5. Add the function call to `RegisterAll` in `tools.go` if it's a new group

```go
srv.RegisterTool(server.Tool{
    Name:        "hudl_resource_action",
    Description: "What it does. When to use it.",
    InputSchema: server.ObjectSchema("", map[string]any{
        "id": server.StringProp("Resource ID"),
    }, []string{"id"}),
}, func(args map[string]any) (any, error) {
    id := server.ArgString(args, "id")
    raw, err := cloudRequest("GET", "/resources/"+id, nil, nil, false)
    if err != nil {
        return nil, wrapError(err)
    }
    return extractKey(raw, "resource"), nil
})
```

### Schema Helpers

| Helper | Produces |
|--------|----------|
| `StringProp(desc)` | `{"type": "string", "description": "..."}` |
| `IntProp(desc)` | `{"type": "integer", "description": "..."}` |
| `BoolProp(desc)` | `{"type": "boolean", "description": "..."}` |
| `StringArrayProp(desc)` | `{"type": "array", "items": {"type": "string"}, ...}` |
| `EnumProp(desc, values)` | `{"type": "string", "enum": [...], ...}` |
| `ObjectSchema(desc, props, required)` | Full object schema with properties and required list |

### Testing

```sh
# Build
make build-mcp

# Smoke test — initialize
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./hudl-mcp 2>/dev/null

# List tools
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | ./hudl-mcp 2>/dev/null | python3 -c "
import sys, json
tools = json.load(sys.stdin)['result']['tools']
print(f'{len(tools)} tools')
for t in tools:
    print(f'  {t[\"name\"]}: {t[\"description\"][:60]}')
"

# Call a tool
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hudl_auth_status","arguments":{}}}' | ./hudl-mcp 2>/dev/null | python3 -m json.tool
```
