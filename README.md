# llmconf - LLM Provider Configuration CLI

A Go-based CLI tool for managing LLM provider configurations across multiple agentic coding tools, starting with Claude Code. Supports both global (user) and project scopes with secure credential storage.

## Features

- **Intelligent Flows**: All commands detect state and guide users through missing configuration automatically
- **No Shell RC Modification**: Everything lives in Claude Code settings.json files
- **Secure by Default**: Credentials stored in OS keychain/keyring, never in JSON files
- **Git-Friendly**: Project settings.json can be committed without secrets
- **Multiple Providers**: Support for Anthropic, Amazon Bedrock, Google Vertex AI, Microsoft Foundry, Fireworks AI, and LiteLLM Proxy

## Installation

### Using go install (requires Go installed)

```bash
go install github.com/stuckinforloop/llmconf@latest
```

### Using pre-built binaries

Download from the [releases page](https://github.com/stuckinforloop/llmconf/releases):

```bash
# macOS (ARM64)
curl -L https://github.com/stuckinforloop/llmconf/releases/latest/download/llmconf_Darwin_arm64.tar.gz | tar xz

# macOS (Intel)
curl -L https://github.com/stuckinforloop/llmconf/releases/latest/download/llmconf_Darwin_x86_64.tar.gz | tar xz

# Linux
curl -L https://github.com/stuckinforloop/llmconf/releases/latest/download/llmconf_Linux_x86_64.tar.gz | tar xz

# Move to PATH
mv llmconf /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/stuckinforloop/llmconf.git
cd llmconf
make build
make install
```

## Quick Start

```bash
# Initialize (intelligent interactive flow)
llmconf init

# Set provider (auto-configures if missing)
llmconf set bedrock --project

# Check status with recommendations
llmconf status

# Rotate credentials
llmconf rotate bedrock

# Diagnose and fix issues
llmconf doctor
```

## Commands

| Command | Description |
|---------|-------------|
| `init` | First-time setup or add new configuration |
| `set <provider>` | Switch to specified provider with auto-configuration |
| `list` | Show all providers and their status |
| `status [tool]` | Show current configuration with recommendations |
| `rotate <provider>` | Rotate credentials with auto-setup detection |
| `remove <provider>` | Remove provider configuration |
| `doctor` | Diagnose and fix configuration issues |
| `config view` | View internal configuration |
| `config path` | Show configuration file paths |
| `version` | Show version information |

## Supported Providers

### 1. Anthropic (Direct API)
- Required: `ANTHROPIC_API_KEY`
- No model pinning required

### 2. Amazon Bedrock
- Required: `CLAUDE_CODE_USE_BEDROCK=1`, `AWS_REGION`
- Auth: SSO profile, API keys, or Bearer token
- Model pinning strongly recommended

### 3. Google Vertex AI
- Required: `CLAUDE_CODE_USE_VERTEX=1`, `CLOUD_ML_REGION`, `ANTHROPIC_VERTEX_PROJECT_ID`
- Model pinning strongly recommended

### 4. Microsoft Foundry
- Required: `CLAUDE_CODE_USE_FOUNDRY=1`, `ANTHROPIC_FOUNDRY_RESOURCE`
- Model pinning strongly recommended

### 5. Fireworks AI
- Required: `ANTHROPIC_BASE_URL`, `ANTHROPIC_API_KEY`
- All model vars must be set to the same Fireworks model

### 6. LiteLLM Proxy
- Required: `ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`

## Configuration Files

- `~/.config/llmconf/config.json` - llmconf internal configuration
- `~/.claude/settings.json` - Global Claude Code settings
- `./.claude/settings.json` - Project scope settings
- `./.claude/settings.local.json` - Local scope settings (detected but not managed)

## Security

- Credentials are never stored in settings.json files
- Project settings.json can be safely committed to git
- OS keychain/keyring integration for secure credential storage
- Key rotation support

## Development

```bash
# Build
make build

# Test
make test

# Run with hot reload (requires air)
make dev

# Build for all platforms
make build-all
```

## License

MIT License - see LICENSE file for details.
