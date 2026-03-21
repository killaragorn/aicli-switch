# aicli-switch: AI CLI 工具凭证切换器

## 概述

轻量级 Go CLI 工具，用于在多个 Claude Code OAuth 账号之间快速切换，无需重新进行 OAuth 认证。支持 OAuth 和 API Key 两种账号类型，OAuth token 过期时自动刷新。

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

## CLI 命令

| 命令 | 说明 |
|------|------|
| `aicli-switch add <name>` | 添加 profile（从当前登录状态导入，或交互式输入 API Key） |
| `aicli-switch rm <name>` | 删除 profile |
| `aicli-switch ls` | 列出所有 profile，标记当前激活项和 token 状态 |
| `aicli-switch <name>` | 切换到指定 profile |
| `aicli-switch status` | 显示当前 profile 详细信息 |
| `aicli-switch refresh [name]` | 手动刷新 OAuth token |

## 切换流程

```
aicli-switch <name>
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
- 密钥：`auth.v2.key` = base64 编码的 32 字节 AES-256-GCM 密钥（16 字节 nonce）
- 明文：`{ "access_token": "eyJ...", "refresh_token": "Fpr..." }`

### 刷新端点

- **URL**: `https://platform.claude.com/v1/oauth/token`
- **Method**: POST
- **Body**: `{"grant_type":"refresh_token","refresh_token":"xxx","client_id":"9d1c250a-e61b-44d9-88ed-5944d1962f5e"}`

## 技术选型

| 项目 | 选择 | 原因 |
|------|------|------|
| 语言 | Go 1.24 | 单二进制，零运行时依赖 |
| CLI 解析 | os.Args | 命令简单，无需框架 |
| 加密 | crypto/aes + crypto/cipher | 标准库 |
| HTTP | net/http | 标准库，用于 token 刷新 |
| 第三方依赖 | 无 | 全部使用标准库 |

## 文件结构

```
aicli-switch/
  ├── go.mod
  ├── main.go                    # CLI 入口 + 命令分发
  ├── internal/
  │   ├── config/paths.go        # 路径常量
  │   ├── crypto/aes.go          # AES-256-GCM 加解密
  │   ├── token/token.go         # JWT 解析、过期检测、token 刷新
  │   ├── profile/profile.go     # Profile CRUD
  │   └── switcher/switcher.go   # 核心切换逻辑
  └── docs/
      └── 2026-03-21-aicli-switch-design.md
```

## 安全考虑

- OAuth 凭证始终以 AES-256-GCM 加密存储，与 Claude Code 原生行为一致
- 每个 profile 使用独立的加密密钥
- refresh_token 不以明文形式暴露在任何配置文件中
- `_active` 文件仅存储 profile 名称，不含敏感信息
