# RiskOS CLI

A Go CLI for integrating with the [RiskOS](https://riskos.socure.com) platform.
It covers authentication, use-case discovery, integration guide generation,
webhook management, and sandbox test execution.

## Build

```bash
make build          # produces build/riskos
make cross          # builds for darwin/linux/windows
```

Requires Go 1.22+.

## Auth model

Two separate Bearer tokens are needed:

| Key | Env var | Used for |
|-----|---------|----------|
| CLI/MCP key | `RISKOS_API_KEY` | All `/api/mcp/*` endpoints |
| Evaluation key | `RISKOS_EVAL_KEY` | `/api/evaluation` (test-case runs) |

### Credential storage

Credentials are stored securely in the **OS keychain** (macOS Keychain, GNOME Keyring, Windows Credential Manager) via [go-keyring](https://github.com/zalando/go-keyring).

On systems without keychain support (CI, containers, headless servers), use environment variables directly — no login needed.

## Commands

### auth

```bash
riskos auth login          # interactive — prompts for CLI/MCP key and optional Eval key
riskos auth status         # show current auth status and credential source
riskos auth logout         # remove credentials from keychain
```

**Login flow:**
1. Prompts for CLI/MCP API key (echo suppressed)
2. Optionally prompts for Evaluation API key
3. Verifies against the RiskOS API
4. Stores in OS keychain on success

### usecase

```bash
riskos usecase list                          # list configured use cases
```

### workflow

```bash
riskos workflow list consumer_onboarding     # list workflows for a use case
```

### asset

```bash
riskos asset guide consumer_onboarding ./guide.md            # integration guide (Markdown)
riskos asset openapi ./openapi.json                          # RiskOS API OpenAPI spec
riskos asset webhook-spec ./webhooks.json                    # Webhooks OpenAPI spec
riskos asset testcases consumer_onboarding ./testcases.json  # test case JSON
```

### webhook

```bash
riskos webhook events                                     # list event names
riskos webhook list                                       # list configured webhooks
riskos webhook create --name my-hook --url https://... \
  --use-case consumer_onboarding \
  --workflow-events evaluation_completed \
  --auth-scheme BEARER_TOKEN --bearer-token-file ./token.txt
riskos webhook update --name my-hook --url https://... --use-case consumer_onboarding
riskos webhook simulate --target http://localhost:3000/webhooks/riskos
```

### testcase

```bash
riskos testcase list                          # list sandbox personas
riskos testcase describe Watchlist_Hit_Reject # show request payload
riskos testcase run Valid_Consumer_Accept     # post to sandbox evaluation API
```

## Environment variables

| Variable | Description |
|----------|-------------|
| `RISKOS_API_KEY` | CLI/MCP Bearer token (overrides keychain credentials) |
| `RISKOS_EVAL_KEY` | Evaluation API Bearer token (overrides keychain credentials) |
| `RISKOS_BASE_URL` | API base URL (overrides keychain; overridden by `--base-url` flag) |
| `RISKOS_QUIET` | Set to `1` to suppress informational output |
| `NO_COLOR` | Disable colour output |

**CI / headless usage:** Skip `auth login` entirely and set `RISKOS_API_KEY` (and optionally `RISKOS_EVAL_KEY`) as environment variables. All commands work without the keychain.

## Global flags

| Flag | Description |
|------|-------------|
| `--base-url <url>` | RiskOS API base URL (default: `https://riskos.sandbox.socure.com`). Use `https://riskos.socure.com` for production. |
| `--debug` | Print HTTP request/response details |
| `-o, --output` | Output format: `plain` or `json` |
| `-q, --quiet` | Suppress informational output |
