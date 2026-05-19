# wealthsimple-finances

Personal scripts for pulling Wealthsimple data into a monthly finances spreadsheet. Uses the unofficial [`ws-api`](https://github.com/gboudreau/ws-api-python) GraphQL client.

## Setup

```bash
uv sync
```

## Usage

```bash
# Inspect your accounts (run this first to find the credit card)
uv run python fetch_transactions.py --list-accounts

# Fetch previous calendar month's credit card transactions as TSV
# (default: settled purchases only; columns: date, description, amount)
uv run python fetch_transactions.py

# Custom date range
uv run python fetch_transactions.py --since 2026-04-01 --until 2026-04-30

# Include payments, refunds, and pending transactions
uv run python fetch_transactions.py --all-transactions

# Add currency, type, and subType columns
uv run python fetch_transactions.py --all-columns

# Pipe straight to clipboard for pasting into Google Sheets
uv run python fetch_transactions.py | pbcopy

# Clear cached session (forces re-login)
uv run python fetch_transactions.py --logout
```

On first run you'll be prompted for email, password, and a 2FA code. The session token is stored in the macOS keychain under service `wealthsimple-finances`; subsequent runs auto-refresh and skip the prompts.
