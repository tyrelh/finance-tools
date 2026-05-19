package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRepoRoot_LocatesPyprojectToml(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "pyproject.toml")); err != nil {
		t.Errorf("returned root %q has no pyproject.toml: %v", root, err)
	}
	if _, err := os.Stat(filepath.Join(root, "scripts", "fetch_transactions.py")); err != nil {
		t.Errorf("returned root %q has no scripts/fetch_transactions.py: %v", root, err)
	}
}
