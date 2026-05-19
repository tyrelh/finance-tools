package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type runFinishedMsg struct {
	Stdout string
	Stderr string
	Err    error
}

type authCheckMsg struct {
	Ok      bool
	Message string
}

func runAuthCheck() tea.Cmd {
	return func() tea.Msg {
		root, err := findRepoRoot()
		if err != nil {
			return authCheckMsg{Ok: false, Message: err.Error()}
		}
		cmd := exec.Command("uv", "run", "python", filepath.Join("scripts", "ws_auth.py"), "--check")
		cmd.Dir = root
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		runErr := cmd.Run()
		message := strings.TrimSpace(stdout.String())
		if message == "" {
			message = strings.TrimSpace(stderr.String())
		}
		if message == "" && runErr != nil {
			message = runErr.Error()
		}
		return authCheckMsg{Ok: runErr == nil, Message: message}
	}
}

func findRepoRoot() (string, error) {
	if dir, err := os.Getwd(); err == nil {
		if root := walkUpForPyproject(dir); root != "" {
			return root, nil
		}
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			if root := walkUpForPyproject(filepath.Dir(resolved)); root != "" {
				return root, nil
			}
		}
	}
	return "", errors.New("could not locate repo root (no pyproject.toml found above CWD or binary)")
}

func walkUpForPyproject(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func runScript(scriptFile string, args []string, stdin string) tea.Cmd {
	return func() tea.Msg {
		root, err := findRepoRoot()
		if err != nil {
			return runFinishedMsg{Err: err}
		}
		scriptPath := filepath.Join("scripts", scriptFile)
		cmdArgs := append([]string{"run", "python", scriptPath}, args...)
		cmd := exec.Command("uv", cmdArgs...)
		cmd.Dir = root

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if stdin != "" {
			cmd.Stdin = strings.NewReader(stdin)
		}

		runErr := cmd.Run()
		return runFinishedMsg{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
			Err:    runErr,
		}
	}
}
