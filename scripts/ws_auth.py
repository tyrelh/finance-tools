"""Wealthsimple authentication: keychain-backed session storage and interactive login."""

import argparse
import json
import sys
from getpass import getpass

import keyring
from ws_api import WealthsimpleAPI
from ws_api.exceptions import (
    CurlException,
    LoginFailedException,
    ManualLoginRequired,
    OTPRequiredException,
    WSApiException,
)
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


def load_or_login() -> tuple[WSAPISession, str]:
    """Return a session from the keychain, or prompt for one if none is saved."""
    stored = load_session()
    if stored is not None:
        return stored
    return interactive_login()


def stdin_login() -> tuple[WSAPISession, str]:
    """Log in non-interactively using JSON credentials from stdin.

    Expects {"email": "...", "password": "...", "otp": "..."} on stdin.
    The otp field is optional; if absent or empty and 2FA is required, raises.
    """
    try:
        creds = json.loads(sys.stdin.read())
    except json.JSONDecodeError as e:
        raise SystemExit(f"Invalid JSON on stdin: {e}")
    email = (creds.get("email") or "").strip()
    password = creds.get("password") or ""
    otp = (creds.get("otp") or "").strip() or None
    if not email or not password:
        raise SystemExit("stdin JSON must include non-empty 'email' and 'password'.")
    session = WealthsimpleAPI.login(
        username=email,
        password=password,
        otp_answer=otp,
        persist_session_fct=persist_session,
    )
    return session, email


def check_session() -> tuple[bool, str]:
    """Hit the API to confirm the saved session is active. Returns (ok, message).

    Refreshes the access token via the refresh token if needed and re-persists.
    """
    stored = load_session()
    if stored is None:
        return False, "No session saved."
    session, username = stored
    try:
        WealthsimpleAPI.from_token(session, persist_session, username)
    except ManualLoginRequired:
        return False, f"Session is invalid or expired for {username}. Run --logout, then log in again."
    except CurlException as e:
        return False, f"Could not reach Wealthsimple to validate session: {e}"
    except WSApiException as e:
        return False, f"Could not validate session for {username}: {e}"
    return True, f"Session is valid for {username}."


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description=__doc__)
    g = p.add_mutually_exclusive_group()
    g.add_argument("--logout", action="store_true", help="Clear saved session from keychain")
    g.add_argument("--status", action="store_true", help="Show whether a session is saved (no network call)")
    g.add_argument("--check", action="store_true", help="Hit the API to confirm the saved session is still active")
    g.add_argument("--from-stdin", action="store_true", help="Read {email,password,otp} JSON from stdin and log in non-interactively (overwrites any saved session)")
    return p.parse_args()


def main() -> int:
    args = parse_args()

    if args.logout:
        clear_session()
        print("Session cleared.", file=sys.stderr)
        return 0

    if args.status:
        stored = load_session()
        if stored is None:
            print("No session saved.")
            return 1
        _, username = stored
        print(f"Session saved for {username}.")
        return 0

    if args.check:
        ok, message = check_session()
        print(message)
        return 0 if ok else 1

    if args.from_stdin:
        try:
            _, username = stdin_login()
        except OTPRequiredException:
            print("2FA code required but none provided.", file=sys.stderr)
            return 1
        except LoginFailedException as e:
            print(f"Login failed: {e}", file=sys.stderr)
            return 1
        print(f"Logged in as {username}.", file=sys.stderr)
        return 0

    if load_session() is not None:
        print("Session already saved. Use --logout to clear it first.", file=sys.stderr)
        return 0

    _, username = interactive_login()
    print(f"Logged in as {username}.", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
