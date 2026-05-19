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

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("could not locate repo root (no pyproject.toml found in any parent directory)")
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
