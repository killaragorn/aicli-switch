# aicli-switch

Switch between multiple Claude Code accounts without re-authenticating.

When one account hits its usage limit, just `aicli-switch another-account` — no need to `claude login` again.

## Install

### npm (recommended)

```bash
npm install -g @kio_ai/aicli-switch
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

Requires Go 1.24+:

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
| `aicli-switch add <name>` | Save current OAuth session as a named profile (default) |
| `aicli-switch add <name> -t apikey` | Add an API Key profile (interactive prompt) |
| `aicli-switch <name>` | Switch to a profile (cleans up all auth conflicts) |
| `aicli-switch ls` | List all profiles with subscription, status and expiry |
| `aicli-switch status` | Show active profile and compare with live credentials |
| `aicli-switch refresh [name]` | Manually refresh an OAuth token |
| `aicli-switch rm <name>` | Delete a profile |
| `aicli-switch version` | Show version |

## What It Looks Like

```
$ aicli-switch ls
  NAME       TYPE    SUBSCRIPTION  STATUS   EXPIRY
▶ work       oauth   max           valid    6.9h
  personal   oauth   pro           valid    4.2h
  relay      apikey  api key       ready    n/a

$ aicli-switch personal
Switched to profile "personal" (oauth)

$ aicli-switch status
Active Profile: personal (oauth)
  Created:  2026-03-21T10:00:00+08:00
  Switched: 2026-03-23T14:30:00+08:00

Profile (saved):
  Token:        valid (expires in 4.2h, at 2026-03-23 18:30:00)
  Subscription: pro

~/.claude/.credentials.json (live):
  Token:        valid (expires in 4.2h, at 2026-03-23 18:30:00)
  Subscription: pro

  ✓ In sync
```

## How It Works

### OAuth Profiles

1. **`add`** — reads `claudeAiOauth` from `~/.claude/.credentials.json` and saves it to `~/.cc-profiles/<name>/oauth.json`
2. **`switch`** — writes saved OAuth data back to `~/.claude/.credentials.json`, clears conflicting API keys from `settings.json`, and removes cached `oauthAccount` from `~/.claude.json` so Claude CLI picks up the new identity
3. **Token refresh** — calls Claude's OAuth endpoint with the saved refresh_token, no browser needed

### API Key Profiles

1. **`add -t apikey`** — prompts for API Key and Base URL, saves to profile
2. **`switch`** — updates `ANTHROPIC_API_KEY` and `ANTHROPIC_BASE_URL` in `~/.claude/settings.json`, clears `claudeAiOauth` from credentials, and removes cached `oauthAccount` from `~/.claude.json`

### What gets cleaned up on switch

Switching profiles ensures a clean auth state by clearing conflicting credentials:

| Switch to | Actions |
|-----------|---------|
| **OAuth** | Write `claudeAiOauth` → clear `ANTHROPIC_API_KEY` from settings → clear `oauthAccount` cache |
| **API Key** | Write `ANTHROPIC_API_KEY` to settings → clear `claudeAiOauth` from credentials → clear `oauthAccount` cache |

This prevents the "Both a token and an API key are set" conflict in Claude Code.

### Storage Layout

```
~/.cc-profiles/
├── _active                # Name of the currently active profile
├── work/
│   ├── oauth.json         # OAuth credentials (accessToken, refreshToken, expiresAt, etc.)
│   └── profile.json       # Metadata (name, type, timestamps)
└── relay/
    ├── settings.env.json  # API Key and Base URL
    └── profile.json
```

### Files Modified on Switch

| File | Purpose |
|------|---------|
| `~/.claude/.credentials.json` | OAuth tokens (`claudeAiOauth`), preserves `mcpOAuth` |
| `~/.claude/settings.json` | API key env vars (`ANTHROPIC_API_KEY`, `ANTHROPIC_BASE_URL`) |
| `~/.claude.json` | Cached account info (`oauthAccount` — email, org, billing) |

## Update Detection

aicli-switch checks for new versions on every startup (non-blocking). When an update is available:

```
Update available: 0.2.0 → 0.3.0
  Run: npm update -g @kio_ai/aicli-switch
  Or:  https://github.com/killaragorn/aicli-switch/releases/tag/v0.3.0
```

## Roadmap

- [ ] OpenAI Codex CLI credential switching
- [ ] Gemini CLI credential switching
- [ ] Automatic failover on rate-limit
- [ ] Profile import/export

## License

MIT
