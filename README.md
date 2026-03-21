# aicli-switch

CLI tool for switching AI CLI tool credentials — seamlessly manage multiple Claude Code OAuth accounts and API Keys without re-authentication.

## Why

When you hit the usage limit on one Claude Code account, you need to switch to another. Normally this means running `claude login` again and going through the full OAuth flow. **aicli-switch** saves your credentials and lets you switch with a single command.

## Features

- **OAuth account switching** — save multiple Claude Code OAuth sessions, switch without re-login
- **Automatic token refresh** — expired tokens are refreshed automatically using the refresh_token
- **API Key profiles** — also supports switching between different API Key + Base URL combos
- **AES-256-GCM encryption** — credentials stored with the same encryption Claude Code uses
- **Zero dependencies** — single Go binary, no runtime required

## Install

```bash
# Build from source
go build -o aicli-switch.exe .

# Copy to PATH
cp aicli-switch.exe ~/.local/bin/
```

## Quick Start

```bash
# 1. Log into your first account (normal claude login)
# 2. Save it as a profile
aicli-switch add work

# 3. Log into another account
claude login
# 4. Save that too
aicli-switch add personal

# 5. Switch anytime
aicli-switch work
aicli-switch personal
```

## Commands

```
aicli-switch add <name> [--type oauth|apikey]   Add a new profile
aicli-switch rm <name>                          Remove a profile
aicli-switch ls                                 List all profiles
aicli-switch <name>                             Switch to a profile
aicli-switch status                             Show current profile info
aicli-switch refresh [name]                     Manually refresh OAuth token
aicli-switch help                               Show help
```

## Example Output

```
$ aicli-switch ls
  NAME       TYPE   EMAIL                 STATUS   EXPIRY
▶ work       oauth  user@company.com      valid    6.9d
  personal   oauth  me@gmail.com          valid    4.2d
  relay      apikey -                     ready    n/a

$ aicli-switch personal
Switched to profile "personal" (oauth)
  Email: me@gmail.com

$ aicli-switch status
Active Profile: personal
  Type:    oauth
  Email:   me@gmail.com
  Token:   valid (expires in 4.2d)
```

## How It Works

1. `aicli-switch add` copies your current OAuth credentials (`~/.factory/auth.v2.*`) into `~/.cc-profiles/<name>/`
2. `aicli-switch <name>` restores that profile's credentials back to `~/.factory/`, checking token expiry first
3. If the token is expired, it automatically refreshes via Claude's OAuth endpoint before switching
4. For API Key profiles, it updates `ANTHROPIC_API_KEY` and `ANTHROPIC_BASE_URL` in `~/.claude/settings.json`

## Storage

```
~/.cc-profiles/
├── _active              # Current active profile name
├── work/
│   ├── auth.v2.file     # Encrypted OAuth tokens
│   ├── auth.v2.key      # AES-256-GCM key
│   └── profile.json     # Profile metadata
└── personal/
    └── ...
```

## Roadmap

- [ ] Support for OpenAI Codex CLI credentials
- [ ] Support for Gemini CLI credentials
- [ ] Automatic failover when rate-limited
- [ ] Profile import/export

## License

MIT
