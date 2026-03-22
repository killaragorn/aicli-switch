# aicli-switch

Switch between multiple Claude Code accounts without re-authenticating.

When one account hits its usage limit, just `aicli-switch another-account` — no need to `claude login` again.

## Install

### npm (recommended)

```bash
npm install -g aicli-switch
```

This downloads the pre-built binary for your platform automatically.

### Manual download

Download the binary for your OS from [Releases](https://github.com/killaragorn/aicli-switch/releases/latest):

| Platform | File |
|----------|------|
| Windows x64 | `aicli-switch-windows-amd64.exe` |
| macOS x64 | `aicli-switch-darwin-amd64` |
| macOS Apple Silicon | `aicli-switch-darwin-arm64` |
| Linux x64 | `aicli-switch-linux-amd64` |
| Linux ARM64 | `aicli-switch-linux-arm64` |

Move it to a directory in your PATH:

```bash
# macOS / Linux
chmod +x aicli-switch-*
mv aicli-switch-* /usr/local/bin/aicli-switch

# Windows — move to any directory in your PATH
move aicli-switch-windows-amd64.exe %USERPROFILE%\.local\bin\aicli-switch.exe
```

### Build from source

Requires Go 1.22+:

```bash
git clone https://github.com/killaragorn/aicli-switch.git
cd aicli-switch
go build -o aicli-switch .
```

## Quick Start

```bash
# Step 1: You're logged into account A in Claude Code
# Save it as a profile
aicli-switch add work

# Step 2: Log into account B
claude login

# Step 3: Save account B
aicli-switch add personal

# Step 4: Now switch freely — no re-login needed
aicli-switch work        # → switches to work account
aicli-switch personal    # → switches to personal account
```

## Commands

| Command | Description |
|---------|-------------|
| `aicli-switch add <name>` | Save current Claude Code session as a named profile |
| `aicli-switch add <name> -t apikey` | Add an API Key profile (interactive) |
| `aicli-switch <name>` | Switch to a profile |
| `aicli-switch ls` | List all profiles with status |
| `aicli-switch status` | Show active profile details |
| `aicli-switch refresh [name]` | Manually refresh an OAuth token |
| `aicli-switch rm <name>` | Delete a profile |
| `aicli-switch version` | Show version |

## What It Looks Like

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

### OAuth Profiles

1. **`add`** — copies your encrypted OAuth credentials (`~/.factory/auth.v2.*`) into `~/.cc-profiles/<name>/`
2. **`switch`** — checks if the token is expired → refreshes if needed → restores credentials to `~/.factory/`
3. **Token refresh** — calls Claude's OAuth endpoint with the saved refresh_token, no browser needed

### API Key Profiles

1. **`add -t apikey`** — prompts for API Key and Base URL, saves to profile
2. **`switch`** — updates `ANTHROPIC_API_KEY` and `ANTHROPIC_BASE_URL` in `~/.claude/settings.json`

### Storage Layout

```
~/.cc-profiles/
├── _active                # Name of the currently active profile
├── work/
│   ├── auth.v2.file       # AES-256-GCM encrypted OAuth tokens
│   ├── auth.v2.key        # Encryption key
│   └── profile.json       # Metadata (name, type, email, timestamps)
└── personal/
    └── ...
```

### Security

- Credentials are **always encrypted** (AES-256-GCM), same as Claude Code itself
- Each profile has its own encryption key
- No plaintext tokens anywhere on disk
- The `_active` file only stores the profile name

## Update Detection

aicli-switch checks for new versions once per day (non-blocking). When an update is available:

```
Update available: 0.1.0 → 0.2.0
  Run: npm update -g aicli-switch
  Or:  https://github.com/killaragorn/aicli-switch/releases/tag/v0.2.0
```

## Roadmap

- [ ] OpenAI Codex CLI credential switching
- [ ] Gemini CLI credential switching
- [ ] Automatic failover on rate-limit
- [ ] Profile import/export

## License

MIT
