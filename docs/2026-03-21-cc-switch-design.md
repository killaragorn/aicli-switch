# cc-switch: Claude Code OAuth 账号切换 CLI 工具

## 概述

轻量级 CLI 工具，用于在多个 Claude Code OAuth 账号之间快速切换，无需重新进行 OAuth 认证。支持 OAuth 和 API Key 两种账号类型，OAuth token 过期时自动刷新。

## 核心需求

1. 预先登录多个账号，保存凭证到独立 profile
2. 一行命令切换账号，无需重新 `claude login`
3. OAuth token 过期时自动用 refresh_token 刷新
4. 同时支持 OAuth 和 API Key 类型账号

## 数据存储

```
~/.cc-profiles/
  ├── _active                    # 文本文件，当前激活的 profile 名
  ├── work/
  │   ├── auth.v2.file           # 加密的 OAuth token（从 ~/.factory/ 复制）
  │   ├── auth.v2.key            # AES-256-GCM 密钥
  │   ├── profile.json           # 元信息
  │   └── settings.env.json      # env 配置片段（API Key 类型使用）
  └── personal/
      └── ...
```

### profile.json

```json
{
  "name": "work",
  "type": "oauth",
  "email": "user@company.com",
  "created_at": "2026-03-21T12:00:00Z",
  "last_switched": "2026-03-21T12:00:00Z"
}
```

### settings.env.json（API Key 类型）

```json
{
  "ANTHROPIC_API_KEY": "sk-xxx",
  "ANTHROPIC_BASE_URL": "https://api.anthropic.com/"
}
```

## CLI 命令

| 命令 | 说明 |
|------|------|
| `cc-switch add <name>` | 添加 profile（从当前登录状态导入，或交互式输入 API Key） |
| `cc-switch rm <name>` | 删除 profile |
| `cc-switch ls` | 列出所有 profile，标记当前激活项和 token 状态 |
| `cc-switch <name>` | 切换到指定 profile |
| `cc-switch status` | 显示当前 profile 详细信息 |
| `cc-switch refresh [name]` | 手动刷新 OAuth token |

## 切换流程

```
cc-switch <name>
  │
  ├─ 1. 读取当前 _active，备份当前凭证到当前 profile
  │
  ├─ 2. 读取目标 profile
  │     ├─ OAuth 类型：检查 access_token 是否过期
  │     │   ├─ 过期 → 用 refresh_token 刷新
  │     │   │   ├─ 刷新成功 → 更新 profile 中的 token
  │     │   │   └─ 刷新失败 → 提示 claude login 重新认证
  │     │   └─ 未过期 → 继续
  │     └─ API Key 类型：无需检查
  │
  ├─ 3. 复制凭证到目标位置
  │     ├─ OAuth：auth.v2.file + auth.v2.key → ~/.factory/
  │     └─ API Key：合并 env → ~/.claude/settings.json
  │
  ├─ 4. 更新 _active 文件
  │
  └─ 5. 输出切换结果
```

## OAuth Token 刷新

### 加密存储格式

- 文件：`auth.v2.file` = `base64(iv):base64(authTag):base64(ciphertext)`
- 密钥：`auth.v2.key` = base64 编码的 32 字节 AES-256-GCM 密钥
- 明文：`{ "access_token": "eyJ...", "refresh_token": "Fpr..." }`

### 刷新端点

- **URL**: `https://platform.claude.com/v1/oauth/token`
- **Method**: POST
- **Content-Type**: application/json
- **Body**:
  ```json
  {
    "grant_type": "refresh_token",
    "refresh_token": "<refresh_token>",
    "client_id": "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
  }
  ```
- **Response**:
  ```json
  {
    "access_token": "eyJ...",
    "token_type": "Bearer",
    "expires_in": 604800,
    "refresh_token": "new_refresh_token (optional)"
  }
  ```

### 过期检测

1. 解码 JWT access_token 的 payload
2. 读取 `exp` 字段（Unix timestamp）
3. 若 `exp - now < 300`（5分钟内过期）→ 触发刷新

### 加密写回

1. 生成 16 字节随机 IV
2. 用原 key + 新 IV 进行 AES-256-GCM 加密
3. 写入格式：`base64(iv):base64(authTag):base64(ciphertext)`

## 技术选型

| 项目 | 选择 | 原因 |
|------|------|------|
| 语言 | Node.js | 系统已安装，无需额外依赖 |
| CLI 解析 | 纯 process.argv | 命令简单，无需框架 |
| 加密 | node:crypto | 内置模块 |
| HTTP | node 内置 fetch | 用于 token 刷新 |
| 安装 | ~/.local/bin/cc-switch | 与 claude 同目录 |

## 文件结构

```
cc-oau-switch/
  ├── bin/
  │   └── cc-switch.js           # CLI 入口（#!/usr/bin/env node）
  ├── lib/
  │   ├── crypto.js              # AES-256-GCM 加解密
  │   ├── token.js               # JWT 解析、过期检测、token 刷新
  │   ├── profile.js             # profile CRUD 操作
  │   ├── switcher.js            # 核心切换逻辑
  │   └── config.js              # 路径常量
  ├── package.json
  └── docs/
      └── 2026-03-21-cc-switch-design.md
```

## 安全考虑

- OAuth 凭证始终以 AES-256-GCM 加密存储，与 Claude Code 原生行为一致
- 每个 profile 使用独立的加密密钥
- refresh_token 不以明文形式暴露在任何配置文件中
- `_active` 文件仅存储 profile 名称，不含敏感信息
