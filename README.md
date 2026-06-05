# RiskOS CLI

A Go CLI for integrating with the [RiskOS](https://riskos.socure.com) platform.
It covers authentication, use-case discovery, integration guide generation,
webhook management, and sandbox test execution.

## Build

```bash
make build          # produces ./riskos
make cross          # builds for darwin/linux/windows
```

Requires Go 1.22+.

## Auth model

Two separate Bearer tokens are needed:

| Key | Flag / Env var | Used for |
|-----|---------------|----------|
| CLI/MCP key | `--api-key` / `RISKOS_API_KEY` | All `/api/mcp/*` endpoints |
| Evaluation key | `--eval-api-key` / `RISKOS_EVAL_KEY` | `/api/evaluation` (test-case runs) |

## Commands

### auth

```bash
riskos auth login --api-key <key>            # authenticate and save credentials
riskos auth login --api-key <key> --eval-api-key <uuid>
riskos auth whoami                           # show current auth status
riskos auth set-eval-key --api-key <uuid>   # store evaluation key
```

### usecases

```bash
riskos usecases list                         # list configured use cases
riskos usecases list -o json
```

### workflows

```bash
riskos workflows list consumer_onboarding    # list workflows for a use case
```

### asset

```bash
riskos asset guide consumer_onboarding ./guide.md        # integration guide (Markdown)
riskos asset openapi ./openapi.yaml                       # OpenAPI 3.1 spec
riskos asset testcases consumer_onboarding ./tcs.json    # test case JSON
```

### webhook

```bash
riskos webhook events                                     # list event names
riskos webhook list                                       # list configured webhooks
riskos webhook create --name my-hook --url https://... \
  --use-case consumer_onboarding \
  --workflow-events evaluation_completed \
  --auth-scheme BEARER_TOKEN --bearer-token secret
riskos webhook update --name my-hook --url https://... --use-case consumer_onboarding
riskos webhook forward --target http://localhost:3000/webhooks/riskos
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
| `RISKOS_API_KEY` | CLI/MCP Bearer token (overrides saved credentials) |
| `RISKOS_EVAL_KEY` | Evaluation API Bearer token (overrides saved credentials) |
| `RISKOS_QUIET` | Set to `1` to suppress informational output |
| `NO_COLOR` | Disable colour output |
