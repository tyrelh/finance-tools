"""Fetch Wealthsimple credit card transactions as TSV for pasting into Google Sheets."""

import argparse
import sys
from datetime import datetime, timedelta
from getpass import getpass

import keyring
from ws_api import WealthsimpleAPI
from ws_api.exceptions import OTPRequiredException
from ws_api.session import WSAPISession

KEYRING_SERVICE = "wealthsimple-finances"
SESSION_KEY = "session"
USERNAME_KEY = "username"


def persist_session(session_json: str, username: str) -> None:
    keyring.set_password(KEYRING_SERVICE, USERNAME_KEY, username)
    keyring.set_password(KEYRING_SERVICE, SESSION_KEY, session_json)


def load_session() -> tuple[WSAPISession, str] | None:
    username = keyring.get_password(KEYRING_SERVICE, USERNAME_KEY)
    session_json = keyring.get_password(KEYRING_SERVICE, SESSION_KEY)
    if not username or not session_json:
        return None
    return WSAPISession.from_json(session_json), username


def clear_session() -> None:
    for key in (USERNAME_KEY, SESSION_KEY):
        try:
            keyring.delete_password(KEYRING_SERVICE, key)
        except keyring.errors.PasswordDeleteError:
            pass


def interactive_login() -> tuple[WSAPISession, str]:
    print("Logging in to Wealthsimple. Only the session token is stored (in macOS keychain).", file=sys.stderr)
    username = input("Email: ").strip()
    password = getpass("Password: ")
    try:
        session = WealthsimpleAPI.login(
            username=username,
            password=password,
            persist_session_fct=persist_session,
        )
    except OTPRequiredException:
        otp = input("2FA code: ").strip()
        session = WealthsimpleAPI.login(
            username=username,
            password=password,
            otp_answer=otp,
            persist_session_fct=persist_session,
        )
    return session, username


def previous_calendar_month(today: datetime) -> tuple[datetime, datetime]:
    first_of_this_month = today.replace(day=1, hour=0, minute=0, second=0, microsecond=0)
    last_of_prev = first_of_this_month - timedelta(microseconds=1)
    first_of_prev = last_of_prev.replace(day=1, hour=0, minute=0, second=0, microsecond=0)
    return first_of_prev, last_of_prev


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--since", type=datetime.fromisoformat, help="Start date YYYY-MM-DD (default: first day of previous month)")
    p.add_argument("--until", type=datetime.fromisoformat, help="End date YYYY-MM-DD (default: last day of previous month)")
    p.add_argument("--list-accounts", action="store_true", help="Print all accounts and exit")
    p.add_argument("--all-transactions", action="store_true", help="Include non-PURCHASE activities and pending transactions")
    p.add_argument("--logout", action="store_true", help="Clear saved session from keychain")
    return p.parse_args()


def tsv_safe(value) -> str:
    return str(value or "").replace("\t", " ").replace("\n", " ").replace("\r", " ")


def main() -> int:
    args = parse_args()

    if args.logout:
        clear_session()
        print("Session cleared.", file=sys.stderr)
        return 0

    stored = load_session()
    if stored is None:
        session, username = interactive_login()
    else:
        session, username = stored

    ws = WealthsimpleAPI.from_token(session, persist_session, username)
    accounts = ws.get_accounts()

    if args.list_accounts:
        print("id\tunifiedAccountType\tdescription\tcurrency")
        for a in accounts:
            print("\t".join(tsv_safe(a.get(k)) for k in ("id", "unifiedAccountType", "description", "currency")))
        return 0

    cc_accounts = [a for a in accounts if "CREDIT" in (a.get("unifiedAccountType") or "").upper()]
    if not cc_accounts:
        print("No credit card accounts found. Run with --list-accounts to inspect.", file=sys.stderr)
        return 1

    print(f"Found {len(cc_accounts)} credit card account(s):", file=sys.stderr)
    for a in cc_accounts:
        print(f"  - {a.get('unifiedAccountType')} ({a.get('description', '')})", file=sys.stderr)

    start, end = (args.since, args.until) if args.since and args.until else previous_calendar_month(datetime.now())
    print(f"Fetching activities from {start.date()} to {end.date()}...", file=sys.stderr)

    activities = ws.get_activities(
        account_id=[a["id"] for a in cc_accounts],
        start_date=start,
        end_date=end,
        order_by="OCCURRED_AT_ASC",
        load_all=True,
    )

    if not args.all_transactions:
        activities = [
            a for a in activities
            if a.get("subType") == "PURCHASE" and a.get("status") != "authorized"
        ]

    print("date\tdescription\tamount\tcurrency\ttype\tsubType")
    for act in activities:
        occurred = act.get("occurredAt") or ""
        print("\t".join([
            occurred[:10],
            tsv_safe(act.get("description")),
            tsv_safe(act.get("amount")),
            tsv_safe(act.get("currency")),
            tsv_safe(act.get("type")),
            tsv_safe(act.get("subType")),
        ]))

    print(f"Returned {len(activities)} activities.", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
