# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

aicli-switch 是一个轻量级 CLI 工具，用于在多个 Claude Code 账户（OAuth 或 API Key）之间无缝切换。使用 Go 编写，零外部依赖。

## Build & Run

```bash
# 构建
go build -o aicli-switch .

# 运行（Windows 自动生成 .exe）
go build -o aicli-switch.exe .

# 直接运行
go run .
```

Go 1.24+ required. 无外部依赖，全部使用 Go 标准库。

## Architecture

```
main.go                    # CLI 入口 & 命令路由（add/rm/ls/status/refresh/help/version）
                           # 未匹配的参数作为 profile name 触发 switcher.Switch()
internal/
  config/paths.go          # 路径常量 & home 目录工具函数
  profile/profile.go       # Profile CRUD、OAuth 文件读写、活跃 profile 追踪
  switcher/switcher.go     # 核心切换逻辑：备份→刷新→部署→清除冲突认证→更新状态
  token/token.go           # JWT 解析（无验证）、OAuth token 刷新、数据结构
  updater/updater.go       # 非阻塞 GitHub release 版本检查（启动时异步执行）
```

### Key Data Flows

**切换 profile 时** (`switcher.Switch`):
1. 备份当前 profile 的 credentials
2. 如果 OAuth token 过期/即将过期（5min 内），自动刷新
3. 部署目标 profile 的凭证到 Claude Code 目录
4. 清除冲突认证（OAuth↔API Key 互斥）
5. 清除 `~/.claude.json` 中的 `oauthAccount` 缓存

**认证冲突处理**:
- 切换到 OAuth profile → 清除 settings.json 中的 ANTHROPIC_API_KEY
- 切换到 API Key profile → 删除 credentials 中的 claudeAiOauth
- 两种情况都清除 oauthAccount 缓存

### File Locations

Profile 存储: `~/.cc-profiles/<name>/` (profile.json, oauth.json, settings.env.json)

Claude Code 凭证文件:
- `~/.claude/.credentials.json` — OAuth tokens (保留 mcpOAuth 字段用 json.RawMessage)
- `~/.claude/settings.json` — API key 环境变量
- `~/.claude.json` — 缓存的 oauthAccount

### npm Distribution

通过 `npm install -g @kio_ai/aicli-switch` 分发。`scripts/install.js` 在 postinstall 时从 GitHub releases 下载对应平台的预编译二进制。`bin/aicli-switch.js` 是 Node.js wrapper。

## Code Conventions

- 文件权限: 目录 0700, 文件 0600
- 使用 `json.RawMessage` 保留未知 JSON 字段（如 mcpOAuth）
- 错误处理: 关键操作返回描述性 error，非关键操作（如更新检查）静默失败
- 输出使用 ANSI 颜色码，`text/tabwriter` 对齐
- OAuth Client ID: `9d1c250a-e61b-44d9-88ed-5944d1962f5e`
- Token refresh endpoint: `https://platform.claude.com/v1/oauth/token`
