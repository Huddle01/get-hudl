<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/Huddle01/get-hudl/main/get/static/logo.png" />
    <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/Huddle01/get-hudl/main/get/static/logo_light.png" />
    <img alt="Huddle01 Cloud" src="https://raw.githubusercontent.com/Huddle01/get-hudl/main/get/static/logo_light.png" width="60" />
  </picture>
</p>

<h1 align="center">@huddle01/mcp</h1>

<p align="center">
  A production-grade <a href="https://modelcontextprotocol.io">Model Context Protocol</a> server that exposes the full <a href="https://console.huddle01.com">Huddle01 Cloud</a> API as <strong>67 tools</strong> — manage VMs, GPUs, networks, and more from any AI agent.
</p>

<p align="center">
  <a href="https://www.npmjs.com/package/@huddle01/mcp"><img alt="npm version" src="https://img.shields.io/npm/v/%40huddle01/mcp" /></a>
  <a href="https://github.com/Huddle01/get-hudl"><img alt="GitHub" src="https://img.shields.io/github/stars/Huddle01/get-hudl?style=flat" /></a>
  <a href="https://github.com/Huddle01/get-hudl/blob/main/LICENSE"><img alt="License" src="https://img.shields.io/npm/l/%40huddle01/mcp" /></a>
</p>

---

## Quick Start

### 1. Install

```sh
npm install -g @huddle01/mcp
```

The postinstall script automatically downloads the correct binary for your platform (macOS, Linux, Windows — x64 and arm64).

### 2. Get an API Key

Sign in to the [Huddle01 Console](https://console.huddle01.com) and generate an API key.

### 3. Add to Your MCP Client

<details>
<summary><strong>Claude Desktop</strong></summary>

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "huddle01": {
      "command": "hudl-mcp",
      "env": {
        "HUDL_API_KEY": "your-api-key"
      }
    }
  }
}
```

Restart Claude Desktop after saving.

</details>

<details>
<summary><strong>Claude Code (CLI & IDE)</strong></summary>

Add the server with one command:

```sh
# Project-scoped (recommended — saved to .mcp.json)
claude mcp add huddle01 hudl-mcp

# User-scoped (available in all projects)
claude mcp add --scope user huddle01 hudl-mcp
```

Set your API key as an environment variable:

```sh
claude mcp add huddle01 hudl-mcp -e HUDL_API_KEY=your-api-key
```

Verify it's registered:

```sh
claude mcp list
```

</details>

<details>
<summary><strong>OpenAI Codex CLI</strong></summary>

Create or edit `~/.codex/config.yaml`:

```yaml
mcp_servers:
  - name: huddle01
    command: hudl-mcp
    env:
      HUDL_API_KEY: your-api-key
```

Then run Codex — the Huddle01 tools will be available automatically.

</details>

<details>
<summary><strong>Cursor</strong></summary>

Add to `.cursor/mcp.json` in your project root (or global settings):

```json
{
  "mcpServers": {
    "huddle01": {
      "command": "hudl-mcp",
      "env": {
        "HUDL_API_KEY": "your-api-key"
      }
    }
  }
}
```

</details>

<details>
<summary><strong>Windsurf</strong></summary>

Add to your Windsurf MCP config (`~/.codeium/windsurf/mcp_config.json`):

```json
{
  "mcpServers": {
    "huddle01": {
      "command": "hudl-mcp",
      "env": {
        "HUDL_API_KEY": "your-api-key"
      }
    }
  }
}
```

</details>

<details>
<summary><strong>VS Code (Copilot)</strong></summary>

Add to `.vscode/mcp.json` in your workspace:

```json
{
  "servers": {
    "huddle01": {
      "type": "stdio",
      "command": "hudl-mcp",
      "env": {
        "HUDL_API_KEY": "your-api-key"
      }
    }
  }
}
```

Or add via the command palette: `MCP: Add Server` → `stdio` → `hudl-mcp`.

</details>

<details>
<summary><strong>Zed</strong></summary>

Add to your Zed settings (`~/.config/zed/settings.json`):

```json
{
  "context_servers": {
    "huddle01": {
      "command": {
        "path": "hudl-mcp",
        "args": [],
        "env": {
          "HUDL_API_KEY": "your-api-key"
        }
      }
    }
  }
}
```

</details>

<details>
<summary><strong>Continue</strong></summary>

Add to your Continue config (`~/.continue/config.yaml`):

```yaml
mcpServers:
  - name: huddle01
    command: hudl-mcp
    env:
      HUDL_API_KEY: your-api-key
```

</details>

### 4. Authenticate

You can authenticate in three ways (in order of priority):

1. **Environment variable** — set `HUDL_API_KEY` in your MCP client config (shown in examples above)
2. **Config file** — run `hudl auth login --token <key>` (writes to `~/.hudl/config.toml`)
3. **MCP tool** — call `hudl_login` with your API key from within the AI chat

---

## Using with Custom MCP Clients

`hudl-mcp` speaks standard [MCP over stdio](https://modelcontextprotocol.io/docs/concepts/transports#stdio) — any MCP-compatible client can use it.

### Node.js (using `@modelcontextprotocol/sdk`)

```js
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";

const transport = new StdioClientTransport({
  command: "hudl-mcp",
  env: { HUDL_API_KEY: process.env.HUDL_API_KEY },
});

const client = new Client({ name: "my-app", version: "1.0.0" });
await client.connect(transport);

// List all tools
const { tools } = await client.listTools();
console.log(`${tools.length} tools available`);

// Call a tool
const result = await client.callTool({
  name: "hudl_vm_list",
  arguments: {},
});
console.log(result.content[0].text);
```

### Python (using `mcp` SDK)

```python
import asyncio
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

async def main():
    params = StdioServerParameters(
        command="hudl-mcp",
        env={"HUDL_API_KEY": "your-api-key"},
    )
    async with stdio_client(params) as (read, write):
        async with ClientSession(read, write) as session:
            await session.initialize()

            # List tools
            tools = await session.list_tools()
            print(f"{len(tools.tools)} tools available")

            # Call a tool
            result = await session.call_tool("hudl_vm_list", {})
            print(result.content[0].text)

asyncio.run(main())
```

### Raw JSON-RPC (any language)

Spawn `hudl-mcp` and communicate via stdin/stdout using newline-delimited JSON-RPC 2.0:

```bash
# Initialize
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"my-client","version":"1.0.0"}}}' | hudl-mcp
```

```bash
# List tools
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | hudl-mcp
```

```bash
# Call a tool
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"hudl_vm_list","arguments":{}}}' | hudl-mcp
```

Each message must be a single line of JSON followed by a newline. Responses are written to stdout in the same format. Diagnostic logs go to stderr.

---

## What Can You Do?

Once connected, ask your AI assistant things like:

- *"List all my VMs in eu2"*
- *"Create a GPU deployment with an A100 and CUDA image"*
- *"Show me available GPU offers under $2/hr"*
- *"Open port 443 on my web-server security group"*
- *"Attach a floating IP to my VM"*
- *"What regions are available?"*

---

## All 67 Tools

### Authentication & Context

| Tool | Description |
|------|-------------|
| `hudl_login` | Store API key for authentication |
| `hudl_auth_status` | Show auth state (masked key, workspace, region) |
| `hudl_auth_clear` | Remove saved API key |
| `hudl_ctx_show` | Show current workspace and region context |
| `hudl_ctx_use` | Set default workspace |
| `hudl_ctx_region` | Set default region |

### Virtual Machines

| Tool | Description |
|------|-------------|
| `hudl_vm_list` | List VMs in current region |
| `hudl_vm_get` | Get VM details by ID |
| `hudl_vm_create` | Create a new VM |
| `hudl_vm_delete` | Delete a VM |
| `hudl_vm_status` | Get VM status |
| `hudl_vm_action` | Start, stop, reboot, pause, resume, or rebuild a VM |
| `hudl_vm_attach_network` | Attach a private network to a VM |

### Block Storage

| Tool | Description |
|------|-------------|
| `hudl_volume_list` | List all volumes |
| `hudl_volume_get` | Get volume details |
| `hudl_volume_create` | Create a volume |
| `hudl_volume_delete` | Delete a volume |
| `hudl_volume_attach` | Attach volume to a VM |
| `hudl_volume_detach` | Detach volume from a VM |

### Floating IPs

| Tool | Description |
|------|-------------|
| `hudl_fip_list` | List floating IPs |
| `hudl_fip_get` | Get floating IP details |
| `hudl_fip_associate` | Associate floating IP with a VM |
| `hudl_fip_disassociate` | Disassociate floating IP from a VM |

### Security Groups

| Tool | Description |
|------|-------------|
| `hudl_sg_list` | List security groups |
| `hudl_sg_get` | Get security group with rules |
| `hudl_sg_create` | Create a security group |
| `hudl_sg_delete` | Delete a security group |
| `hudl_sg_duplicate` | Duplicate a security group to another region |
| `hudl_sg_rule_add` | Add a firewall rule |
| `hudl_sg_rule_delete` | Delete a firewall rule |

### Networks

| Tool | Description |
|------|-------------|
| `hudl_network_list` | List private networks |
| `hudl_network_create` | Create a private network |
| `hudl_network_delete` | Delete a private network |

### SSH Keys

| Tool | Description |
|------|-------------|
| `hudl_key_list` | List SSH key pairs |
| `hudl_key_get` | Get key pair details |
| `hudl_key_create` | Create a key pair |
| `hudl_key_delete` | Delete a key pair |

### Lookup

| Tool | Description |
|------|-------------|
| `hudl_flavor_list` | List compute flavors |
| `hudl_image_list` | List OS images |
| `hudl_region_list` | List available regions |

### GPU Marketplace

| Tool | Description |
|------|-------------|
| `hudl_gpu_offers` | List GPU marketplace offers with filters |
| `hudl_gpu_summary` | Aggregate GPU availability summary |
| `hudl_gpu_check` | Check cluster type availability |

### GPU Deployments

| Tool | Description |
|------|-------------|
| `hudl_gpu_list` | List GPU deployments |
| `hudl_gpu_get` | Get deployment details |
| `hudl_gpu_deploy` | Deploy a GPU cluster |
| `hudl_gpu_action` | Start, stop, or reboot a deployment |
| `hudl_gpu_delete` | Delete a deployment |

### GPU Waitlist

| Tool | Description |
|------|-------------|
| `hudl_gpu_waitlist_list` | List waitlist requests |
| `hudl_gpu_waitlist_add` | Create a waitlist request with optional auto-deploy |
| `hudl_gpu_waitlist_cancel` | Cancel a waitlist request |

### GPU Images

| Tool | Description |
|------|-------------|
| `hudl_gpu_image_list` | List GPU images with filters |

### GPU Volumes

| Tool | Description |
|------|-------------|
| `hudl_gpu_volume_list` | List GPU volumes |
| `hudl_gpu_volume_create` | Create a GPU volume |
| `hudl_gpu_volume_delete` | Delete a GPU volume |
| `hudl_gpu_volume_type_list` | List GPU volume types |

### GPU SSH Keys

| Tool | Description |
|------|-------------|
| `hudl_gpu_ssh_key_list` | List GPU SSH keys |
| `hudl_gpu_ssh_key_upload` | Upload a GPU SSH key |
| `hudl_gpu_ssh_key_delete` | Delete a GPU SSH key |

### GPU API Keys

| Tool | Description |
|------|-------------|
| `hudl_gpu_api_key_list` | List GPU API keys |
| `hudl_gpu_api_key_create` | Create a GPU API key |
| `hudl_gpu_api_key_revoke` | Revoke a GPU API key |

### GPU Webhooks

| Tool | Description |
|------|-------------|
| `hudl_gpu_webhook_list` | List webhooks |
| `hudl_gpu_webhook_create` | Create a webhook |
| `hudl_gpu_webhook_update` | Update a webhook |
| `hudl_gpu_webhook_delete` | Delete a webhook |

### GPU Regions

| Tool | Description |
|------|-------------|
| `hudl_gpu_region_list` | List available GPU regions |

---

## Configuration

The MCP server reads the same configuration as the [Huddle01 CLI](https://github.com/Huddle01/get-hudl):

| Source | Example |
|--------|---------|
| Config file | `~/.hudl/config.toml` |
| Project config | `./hudl.toml` |
| Environment variables | `HUDL_API_KEY`, `HUDL_WORKSPACE`, `HUDL_REGION` |

```toml
# ~/.hudl/config.toml
[auth]
token = "your-api-key"

[context]
workspace = "my-org"
region = "eu2"
```

---

## Architecture

```
┌───────────────────────────────────────────────┐
│  MCP Client (Claude, Cursor, VS Code, etc.)   │
└──────────────────────┬────────────────────────┘
                       │ JSON-RPC 2.0 over stdio
                       ▼
┌──────────────────────────────────────────────┐
│  hudl-mcp                                    │
│  ┌────────────────────────────────────────┐  │
│  │  MCP protocol engine (server.go)       │  │
│  │  → 67 tool handlers (auth, cloud, GPU) │  │
│  │  → HTTP client with retry + backoff    │  │
│  └────────────────────┬───────────────────┘  │
└───────────────────────┼──────────────────────┘
                        │ HTTPS
                        ▼
              ┌───────────────────┐
              │  Huddle01 Cloud   │
              │  APIs             │
              └───────────────────┘
```

- **Pure Go** binary (~6 MB) with zero external MCP dependencies
- **Stdio transport** — standard MCP communication, works with every client
- **Structured errors** — HTTP errors preserve status code, message, and request ID as JSON
- **Automatic retries** — up to 4 attempts with exponential backoff for transient failures
- **Secure** — API keys stored with `0600` permissions, masked in output

---

## Supported Platforms

| OS | Architecture |
|----|-------------|
| macOS | x64, arm64 |
| Linux | x64, arm64 |
| Windows | x64, arm64 |

---

## Manual Installation

If the automatic postinstall fails, download the binary directly from [GitHub Releases](https://github.com/Huddle01/get-hudl/releases):

```sh
# Example for macOS arm64
curl -L -o hudl-mcp \
  https://github.com/Huddle01/get-hudl/releases/latest/download/hudl-mcp-darwin-arm64
chmod +x hudl-mcp
```

Or build from source:

```sh
git clone https://github.com/Huddle01/get-hudl.git
cd get-hudl
make build-mcp
```

Requires Go 1.25+.

---

## Troubleshooting

### Binary not found after install

If `hudl-mcp` is not on your `PATH`, use the full path to the binary:

```sh
# Find where npm installed it
npm list -g @huddle01/mcp
# Use the full path in your MCP config
npx hudl-mcp  # or: $(npm root -g)/@huddle01/mcp/bin/hudl-mcp
```

### `npx` as a fallback

If global install isn't an option, use `npx` in your MCP client config:

```json
{
  "mcpServers": {
    "huddle01": {
      "command": "npx",
      "args": ["-y", "@huddle01/mcp"],
      "env": {
        "HUDL_API_KEY": "your-api-key"
      }
    }
  }
}
```

### Postinstall fails (restricted network / CI)

Download the binary manually and place it at `node_modules/@huddle01/mcp/bin/hudl-mcp`:

```sh
# Example for Linux x64
curl -L -o node_modules/@huddle01/mcp/bin/hudl-mcp \
  https://github.com/Huddle01/get-hudl/releases/latest/download/hudl-mcp-linux-amd64
chmod +x node_modules/@huddle01/mcp/bin/hudl-mcp
```

### Authentication errors

1. Verify your key is valid: `hudl auth login --token <key> && hudl auth status`
2. Check the config file exists: `cat ~/.hudl/config.toml`
3. Or pass the key via environment variable: `HUDL_API_KEY=<key> hudl-mcp`

### Debug mode

Run the binary directly to see stderr diagnostics:

```sh
HUDL_API_KEY=your-key hudl-mcp 2>debug.log
```

---

## Links

- [Huddle01 Console](https://console.huddle01.com)
- [CLI Documentation](https://console.huddle01.com/docs/cli)
- [GitHub Repository](https://github.com/Huddle01/get-hudl)
- [MCP Protocol Specification](https://modelcontextprotocol.io)

## License

MIT
