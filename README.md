# wealthsimple-finances

Personal scripts for pulling Wealthsimple data into a monthly finances spreadsheet. Uses the unofficial [`ws-api`](https://github.com/gboudreau/ws-api-python) GraphQL client.

## Layout

- `scripts/` — Python scripts that do the actual work
- `tui/` — Go TUI (Bubble Tea) that wraps the scripts in an interactive frontend

## Setup

```bash
uv sync
```

## TUI

A Go-based TUI wraps the scripts:

```bash
cd tui && go run .
```

Pick a script, fill in the form, see results in a scrollable table with copy-to-clipboard. On startup the picker shows a session status line (`✓` green if valid, `✗` red if expired/missing) — so you'll know up front whether you need to re-authenticate. If you do, pick **Log in to Wealthsimple** from the menu.

## Scripts

### `scripts/ws_auth.py`

Handles authentication. Session tokens live in the macOS keychain under service `wealthsimple-finances` — never on disk.

```bash
# Interactive login (prompts for email, password, 2FA code)
uv run python scripts/ws_auth.py

# Show whether a session is saved (offline, no network call)
uv run python scripts/ws_auth.py --status

# Hit the API to confirm the session is still active
# (also refreshes the access token if it's expired but refresh token is good)
uv run python scripts/ws_auth.py --check

# Clear the saved session
uv run python scripts/ws_auth.py --logout

# Non-interactive login (used by the TUI; reads JSON from stdin)
echo '{"email":"...","password":"...","otp":"..."}' \
  | uv run python scripts/ws_auth.py --from-stdin
```

The module is also importable from other scripts in `scripts/` (`load_or_login`, `check_session`, `persist_session`, …).

### `scripts/fetch_transactions.py`

```bash
# Inspect your accounts (run this first to find the credit card)
uv run python scripts/fetch_transactions.py --list-accounts

# Fetch previous calendar month's credit card transactions as TSV
# (default: settled purchases only; columns: date, description, amount)
uv run python scripts/fetch_transactions.py

# Custom date range
uv run python scripts/fetch_transactions.py --since 2026-04-01 --until 2026-04-30

# Include payments, refunds, and pending transactions
uv run python scripts/fetch_transactions.py --all-transactions

# Add currency, type, and subType columns
uv run python scripts/fetch_transactions.py --all-columns

# Pipe straight to clipboard for pasting into Google Sheets
uv run python scripts/fetch_transactions.py | pbcopy

# Clear cached session (forces re-login)
uv run python scripts/fetch_transactions.py --logout
```

On first run you'll be prompted for email, password, and a 2FA code (delegated to `ws_auth.py`). Subsequent runs reuse the keychain session and auto-refresh the access token when needed.
