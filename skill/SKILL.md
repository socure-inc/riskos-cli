---
name: riskos-cli-integration-skill
description: "This skill focuses on Socure RiskOS API integration using the RiskOS CLI"
---

## Objective

Guide the user through a **complete, working integration** with the RiskOS Platform using the `riskos` CLI tool.

The integration is considered successful only when a client application can invoke a **live Sandbox workflow by making an API call**, receive a **RiskOS decision** (Accept, Reject, or Review), and correctly handle the response.

---

## Core Principles

- Integration is the final goal — do not stop at documentation or setup steps
- Progress step-by-step and confirm required inputs before advancing
- Use CLI output and downloaded assets as the **single source of truth**
- Prefer Sandbox-based validation before any Production discussion
- Do not assume configuration, workflows, or environments unless explicitly confirmed
- Always provide documentation links only from https://help.socure.com/riskos to guide the user about RiskOS
- **The agent must NEVER ask for, handle, or view user credentials** — guide the user to enter them interactively

---

## State Management

The skill must explicitly track and confirm the following before proceeding:

- Programming language and framework
- Environment (Sandbox or Production)
- Selected use case
- Selected **live** workflow
- Integration approach:
    - API-only integration
    - Socure Hosted UX

Do **not** advance to later steps until the required state is confirmed.

---

## Authentication

**Critical: The agent must NEVER ask for or handle credentials directly.**

The `riskos` CLI stores credentials securely in the OS keychain (macOS Keychain, GNOME Keyring, Windows Credential Manager). The agent's role is to guide the user through authentication and verify it succeeded — without ever seeing keys.

### Login flow

Instruct the user to run `riskos auth login` themselves. This is an interactive command with echo-suppressed input — credentials go directly to the OS keychain:

```bash
riskos auth login
```

The command will prompt for:
1. CLI/MCP API key (required)
2. Evaluation API key (optional — needed for `testcase run`)

After the user confirms they completed login, verify by running:

```bash
riskos auth status
```

This shows authentication state (status, namespace, credential source) **without exposing keys**.

### Headless / CI environments

If the OS keychain is unavailable, instruct the user to set environment variables themselves:

- `RISKOS_API_KEY` — CLI/MCP Bearer token
- `RISKOS_EVAL_KEY` — Evaluation API Bearer token (optional)

The agent must NOT read these env var values. Verify with `riskos auth status`.

### Logout

```bash
riskos auth logout
```

Removes credentials from keychain. Environment variables (if set) remain active.

### Agent rules for auth

- Do NOT run `riskos auth login` — it requires interactive credential input
- Do NOT suggest piping keys into stdin
- Do NOT read or echo environment variable values containing keys
- DO run `riskos auth status` to verify authentication state
- DO run authenticated commands (`usecase list`, `testcase list`) to confirm access works

---

## Integration Plan

### 1. Set Up the Development Context

Ask the user:

- Which programming language and framework they want to use

Confirm before continuing.

---

### 2. Obtain RiskOS Credentials

Guide the user through authentication:

1. Instruct user to run `riskos auth login` (they enter keys interactively)
2. Verify with `riskos auth status`
3. Confirm the output shows `"status": "authenticated"`

If using CI/headless: instruct user to set `RISKOS_API_KEY` (and optionally `RISKOS_EVAL_KEY`) as environment variables, then verify with `riskos auth status`.

Do not proceed until authentication is confirmed.

---

### 3. Understand Environments

Explain to the user:

- The CLI defaults to the **Sandbox environment** (`https://riskos.sandbox.socure.com`)
- Only **live Sandbox workflows** can be used for integration and evaluation
- Production workflows require `--base-url https://riskos.socure.com`
- Test cases and simulation are Sandbox-only features

Reference documentation at https://help.socure.com/riskos for environment details.

---

### 4. Discover Eligible Use Cases and Workflows

Use the following sequence:

1. Run `riskos usecase list` to retrieve available use cases
2. For each use case, run `riskos workflow list <usecase>` to see configured workflows
3. Identify use cases with at least one **live** workflow

```bash
riskos usecase list
riskos workflow list consumer_onboarding
```

If a use case has **no live workflows**:

- Clearly explain that integration cannot proceed for that use case
- Ask the user to select a different use case

Present only eligible use cases and ask the user to select one.

---

### 5. Select the Integration Path

Once the use case is selected, identify the integration approach:

- **API-only integration** (build your own UI)
- **Socure Hosted UX**

If Socure Hosted UX is selected, reference the Hosted UX integration guide from https://help.socure.com/riskos.

---

### 6. Download and Read Integration Guides

Download the integration guide for the selected use case:

```bash
riskos asset guide <usecase> ./riskos-guide.md
```

Read the downloaded guide file to understand:

- Required inputs and fields
- Decisioning behavior
- Expected responses
- Use-case-specific constraints
- Edge cases and negative scenarios

---

### 7. Work Through the Integration Checklist

Using the downloaded integration guide, work through the checklist **step by step** with the user:

- Explain each integration item
- Confirm completion before moving on
- Adjust steps if Socure Hosted UX is used

Do not skip checklist items. Give preference to the guide content. For additional context, reference https://help.socure.com/riskos.

---

### 8. Understand the RiskOS API Model

Download and review the OpenAPI specification:

```bash
riskos asset openapi ./riskos-openapi.json
```

Use this specification as the **single source of truth** for:

- Request and response schemas
- Required and optional fields
- Enum values
- Error models

Use it to generate or validate integration code. Note: every workflow is unique and the API schema might not fit exactly.

---

### 9. Use Test Cases for Integration

Discover available test cases:

```bash
riskos testcase list <usecase>
riskos testcase describe <usecase> <testcase_name>
```

Download test cases for reference:

```bash
riskos asset testcases <usecase> ./testcases.json
```

Leverage test cases to:

- Understand sample requests and responses
- Populate required fields correctly
- Validate expected decision outcomes
- Plan integration for edge cases and negative scenarios

Test cases use **Sandbox-approved synthetic PII only** — they cannot be used in production.

---

### 10. Implement the Client Application

Guide the user to implement the client application that:

- Authenticates using the RiskOS API key
- Submits requests to the selected live workflow
- Parses and handles the RiskOS response

Clearly explain how to handle the final outcome:

- **Accept** — user passed verification
- **Reject** — user failed verification
- **Review** — manual review required

Ensure the client's behavior aligns with the final RiskOS decision.

---

### 11. Test and Validate the Integration

For each major integration step:

1. Run sandbox test cases to validate end-to-end:

```bash
riskos testcase run Valid_Consumer_Accept
riskos testcase run Watchlist_Hit_Reject
```

2. Simulate webhook deliveries to test the webhook handler:

```bash
riskos webhook simulate --target http://localhost:<port>/webhooks/riskos
```

3. Verify the RiskOS decision and response handling

Iterate until results are correct and stable.

---

### 12. Troubleshoot Errors

If any errors occur:

- Add `--debug` flag to see HTTP request/response details
- Surface the HTTP status code and error message
- Reference https://help.socure.com/riskos for error documentation
- Explain the root cause in clear language
- Propose a concrete fix
- Retry and revalidate

Do not proceed until errors are resolved.

---

### 13. Completion and Next Steps

Once integration is successful:

- Confirm that a live Sandbox workflow is invoked successfully
- Confirm correct handling of Accept, Reject, and Review outcomes

Offer to:

- Walk through additional sample API calls
- Discuss Production readiness considerations
- Discuss edge cases and negative scenarios that need to be handled

---

## Webhook Simulation

The `riskos webhook simulate` command delivers simulated RiskOS webhook events to a local endpoint for development testing.

Event payloads are generated from the official RiskOS OpenAPI webhook specification. All values are synthetic examples that demonstrate the **structure** of each event type — they do not represent real production data.

### Basic usage

Deliver 8 simulated events (default) to a local server:

```bash
riskos webhook simulate --target http://localhost:3000/webhooks/riskos
```

### Single event mode

Send one specific event type:

```bash
riskos webhook simulate --target http://localhost:3000/webhooks/riskos --event evaluation_completed
```

### Dry-run mode

Print payloads without making HTTP requests:

```bash
riskos webhook simulate --dry-run --event decision_updated
```

### Custom event count

```bash
riskos webhook simulate --target http://localhost:3000/webhooks/riskos --events 20
```

### HTTP delivery details

- Method: POST with JSON body
- Headers: `Content-Type: application/json`, `X-RiskOS-Simulated: true`, `X-RiskOS-Event-Type: <type>`
- Timeout: 10 seconds per request
- Delay between events: 1.5–3 seconds (simulates realistic timing)

### Available event types

Discover event types with:

```bash
riskos webhook events
```

Event categories:
- **Workflow events**: `workflow_execution_failed`, `evaluation_completed`, `evaluation_rerun`, `evaluation_paused`
- **Case events**: `decision_updated`, `review_case_assigned`, `review_case_unassigned`, `case_status_updated`, `case_notes_added`, `case_attachment_added`, `fraud_confirming`

### Webhook management

```bash
riskos webhook list                          # list configured webhooks
riskos webhook create --name <n> --url <u> --use-case <uc> --workflow-events <ev>
riskos webhook update --name <n> --url <u> --use-case <uc>
```

---

## Commands Reference

| Command | Purpose |
|---------|---------|
| `riskos auth login` | Interactive credential entry — stores in OS keychain |
| `riskos auth status` | Show auth state (safe to run — never exposes keys) |
| `riskos auth logout` | Remove credentials from keychain |
| `riskos usecase list` | List configured use cases |
| `riskos workflow list <usecase>` | List workflows for a use case |
| `riskos asset guide <usecase> <path>` | Download integration guide (Markdown) |
| `riskos asset openapi <path>` | Download RiskOS API OpenAPI spec |
| `riskos asset webhook-spec <path>` | Download webhooks OpenAPI spec |
| `riskos asset testcases <usecase> <path>` | Download test cases (JSON) |
| `riskos testcase list <usecase>` | List test cases for a use case |
| `riskos testcase describe <usecase> <name>` | Show test case request payload |
| `riskos testcase run <name>` | Execute test case against sandbox (requires eval key) |
| `riskos webhook events` | List available webhook event names |
| `riskos webhook list` | List configured webhooks |
| `riskos webhook create` | Create a webhook endpoint |
| `riskos webhook update` | Update a webhook endpoint |
| `riskos webhook simulate` | Simulate webhook deliveries to local endpoint |

---

## Global Flags

| Flag | Description |
|------|-------------|
| `--base-url <url>` | RiskOS API base URL (default: `https://riskos.sandbox.socure.com`) |
| `--debug` | Print HTTP request/response details |
| `-o, --output` | Output format: `plain` or `json` (default: plain in TTY, json when piped) |
| `-q, --quiet` | Suppress informational output |

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `RISKOS_API_KEY` | CLI/MCP Bearer token (overrides keychain) |
| `RISKOS_EVAL_KEY` | Evaluation API Bearer token (overrides keychain) |
| `RISKOS_BASE_URL` | API base URL (overrides keychain; overridden by `--base-url`) |
| `RISKOS_QUIET` | Set to `1` to suppress informational output |
| `NO_COLOR` | Disable colour output |

---

## Guidelines

- You MUST NOT ask for, handle, read, or display user credentials
- You MUST use `riskos auth status` to verify authentication (never inspect keys)
- You MUST use CLI command output as the source of truth for available use cases, workflows, and test cases
- You MUST read downloaded asset files (guides, specs, test cases) to provide accurate guidance
- You MUST NOT use web search for Socure resources — reference https://help.socure.com/riskos only
- You MUST plan and execute tests after each significant integration change
- If `riskos workflow list` returns no live workflows, remind the user that only live workflows are eligible for integration
- Use `--debug` flag when troubleshooting API issues
- Use `-o json` for programmatic parsing of CLI output
