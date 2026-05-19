package main

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
)

// Script is a registry entry describing one runnable Python script in scripts/.
type Script struct {
	ID          string
	Name        string
	Description string
	New         func() *Invocation
}

// Invocation is a single configured run of a script: its form, the args it
// will produce when the form completes, and the columns its TSV output uses.
// Stdin is optional; when nil or returning "", the child process gets no stdin.
// When Columns returns an empty slice, the runner renders stderr/stdout as
// text instead of a table.
type Invocation struct {
	Form       *huh.Form
	ScriptFile string
	BuildArgs  func() []string
	Columns    func() []string
	Stdin      func() string
}

// Registry of available scripts. Add new entries here as the project grows.
var registry = []Script{
	{
		ID:          "fetch_transactions",
		Name:        "Fetch transactions",
		Description: "Pull Wealthsimple credit card transactions as TSV",
		New:         newFetchTransactions,
	},
	{
		ID:          "ws_auth",
		Name:        "Log in to Wealthsimple",
		Description: "Authenticate and save a session to the macOS keychain",
		New:         newLogin,
	},
}

func newFetchTransactions() *Invocation {
	state := struct {
		Since           string
		Until           string
		ListAccounts    bool
		AllTransactions bool
		AllColumns      bool
	}{}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Fetch transactions").
				Description("Leave dates blank to default to the previous calendar month."),
			huh.NewInput().
				Title("Since (YYYY-MM-DD)").
				Placeholder("blank = first day of previous month").
				Value(&state.Since).
				Validate(validateOptionalDate),
			huh.NewInput().
				Title("Until (YYYY-MM-DD)").
				Placeholder("blank = last day of previous month").
				Value(&state.Until).
				Validate(validateOptionalDate),
			huh.NewConfirm().
				Title("List accounts instead of transactions?").
				Description("Useful for discovering account IDs and types.").
				Value(&state.ListAccounts),
			huh.NewConfirm().
				Title("Include non-purchases & pending?").
				Description("Adds payments, refunds, and pending authorizations.").
				Value(&state.AllTransactions),
			huh.NewConfirm().
				Title("Include extra columns?").
				Description("Adds currency, type, subType.").
				Value(&state.AllColumns),
		),
	)

	return &Invocation{
		Form:       form,
		ScriptFile: "fetch_transactions.py",
		BuildArgs: func() []string {
			if state.ListAccounts {
				return []string{"--list-accounts"}
			}
			var args []string
			if state.Since != "" {
				args = append(args, "--since", state.Since)
			}
			if state.Until != "" {
				args = append(args, "--until", state.Until)
			}
			if state.AllTransactions {
				args = append(args, "--all-transactions")
			}
			if state.AllColumns {
				args = append(args, "--all-columns")
			}
			return args
		},
		Columns: func() []string {
			if state.ListAccounts {
				return []string{"id", "unifiedAccountType", "description", "currency"}
			}
			cols := []string{"date", "description", "amount"}
			if state.AllColumns {
				cols = append(cols, "currency", "type", "subType")
			}
			return cols
		},
	}
}

func validateOptionalDate(s string) error {
	if s == "" {
		return nil
	}
	_, err := time.Parse("2006-01-02", s)
	return err
}

func newLogin() *Invocation {
	state := struct {
		Email    string
		Password string
		OTP      string
	}{}

	required := func(label string) func(string) error {
		return func(s string) error {
			if strings.TrimSpace(s) == "" {
				return errEmptyField(label)
			}
			return nil
		}
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Log in to Wealthsimple").
				Description("Credentials are piped to the script via stdin and never written to disk. Only the session token is persisted (macOS keychain)."),
			huh.NewInput().
				Title("Email").
				Value(&state.Email).
				Validate(required("email")),
			huh.NewInput().
				Title("Password").
				EchoMode(huh.EchoModePassword).
				Value(&state.Password).
				Validate(required("password")),
			huh.NewInput().
				Title("2FA code").
				Placeholder("6-digit TOTP from your authenticator").
				Value(&state.OTP).
				Validate(required("2FA code")),
		),
	)

	return &Invocation{
		Form:       form,
		ScriptFile: "ws_auth.py",
		BuildArgs:  func() []string { return []string{"--from-stdin"} },
		Columns:    func() []string { return nil },
		Stdin: func() string {
			payload, _ := json.Marshal(map[string]string{
				"email":    strings.TrimSpace(state.Email),
				"password": state.Password,
				"otp":      strings.TrimSpace(state.OTP),
			})
			return string(payload)
		},
	}
}

type errEmptyField string

func (e errEmptyField) Error() string { return string(e) + " is required" }
