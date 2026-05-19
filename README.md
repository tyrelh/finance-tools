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

Pick a script, fill in the form, see results in a scrollable table with copy-to-clipboard. **Note:** the TUI assumes you've already logged in once via the CLI (the keychain session must exist) — it doesn't do interactive auth.

## Scripts

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

On first run you'll be prompted for email, password, and a 2FA code. The session token is stored in the macOS keychain under service `wealthsimple-finances`; subsequent runs auto-refresh and skip the prompts.
