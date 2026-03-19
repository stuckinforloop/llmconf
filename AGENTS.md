# llmconf - LLM Provider Configuration CLI

## Project Overview

A Go-based CLI tool for managing LLM provider configurations across multiple agentic coding tools, starting with Claude Code. Supports both global (user) and project scopes with secure credential storage.

## Quick Reference

### Project Structure
```
llmconf/
├── main.go                      # Entry point (root for go install)
├── internal/
│   ├── cli/                      # Cobra CLI commands
│   ├── config/                   # Configuration management
│   ├── providers/                # Provider implementations
│   ├── secrets/                  # Secure credential storage
│   ├── tools/                    # Agentic tool integrations
│   └── validator/                # Configuration validation
└── pkg/models/                   # Public types
```

### Key Commands
- `llmconf init` - First-time intelligent setup
- `llmconf set <provider>` - Switch provider with auto-configuration
- `llmconf list` - Show all providers and status
- `llmconf status` - Current configuration with recommendations
- `llmconf rotate <provider>` - Rotate credentials
- `llmconf doctor` - Diagnose and fix issues

### Provider Support
1. **Anthropic** - Direct API (ANTHROPIC_API_KEY)
2. **Amazon Bedrock** - AWS SSO, API keys, or Bearer token
3. **Google Vertex AI** - GCP project-based
4. **Microsoft Foundry** - Azure resource-based
5. **Fireworks AI** - API key with model pinning
6. **LiteLLM Proxy** - Proxy URL + auth token

### Design Principles
1. **Intelligent Flows** - Detect state and guide users automatically
2. **No Shell RC Mod** - Everything in Claude Code settings.json
3. **Secure by Default** - OS keychain storage, never in JSON files
4. **Git-Friendly** - Project settings.json can be committed without secrets

### Building & Testing
```bash
make build          # Build binary
make test           # Run tests
make test-snapshots # Update snapshot tests
make install        # Install to GOPATH/bin
make clean          # Clean build artifacts
```

### Key Files
- `internal/providers/provider.go` - Provider interface
- `internal/config/scope.go` - Global vs Project scope logic
- `internal/secrets/store.go` - Secret storage interface
- `internal/cli/init.go` - Intelligent init flow
- `test/snapshots/` - go-snaps snapshot files

### Configuration Files
- `~/.claude/settings.json` - Global Claude Code settings
- `./.claude/settings.json` - Project scope settings
- `~/.config/llmconf/config.json` - llmconf internal state

### Dependencies
- Cobra/Viper - CLI framework
- Charm (huh, lipgloss, bubbles) - Interactive prompts
- Zalando go-keyring - OS secret storage
- go-snaps - Snapshot testing
