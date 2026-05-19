package main

import (
	"testing"
)

func TestParseTSV_SkipsMatchingHeader(t *testing.T) {
	raw := "id\tunifiedAccountType\tdescription\tcurrency\nabc-123\tCASH\tChecking\tCAD\n"
	cols := []string{"id", "unifiedAccountType", "description", "currency"}
	rows, tcols := parseTSV(raw, cols)
	if len(rows) != 1 {
		t.Fatalf("want 1 row (header skipped), got %d", len(rows))
	}
	if rows[0][0] != "abc-123" {
		t.Errorf("want first cell 'abc-123', got %q", rows[0][0])
	}
	if len(tcols) != 4 {
		t.Errorf("want 4 columns, got %d", len(tcols))
	}
}

func TestParseTSV_NoHeader(t *testing.T) {
	raw := "2026-04-01\tStarbucks\t-5.75\n2026-04-02\tLoblaws\t-42.10\n"
	cols := []string{"date", "description", "amount"}
	rows, _ := parseTSV(raw, cols)
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if rows[1][1] != "Loblaws" {
		t.Errorf("want 'Loblaws' in row 1, got %q", rows[1][1])
	}
}

func TestParseTSV_PadsShortRows(t *testing.T) {
	raw := "2026-04-01\tStarbucks\n"
	cols := []string{"date", "description", "amount"}
	rows, _ := parseTSV(raw, cols)
	if len(rows) != 1 || len(rows[0]) != 3 {
		t.Fatalf("want 1 row of 3 fields, got %d rows of %d fields", len(rows), len(rows[0]))
	}
	if rows[0][2] != "" {
		t.Errorf("want empty amount, got %q", rows[0][2])
	}
}

func TestParseTSV_TrimsExcessFields(t *testing.T) {
	raw := "a\tb\tc\td\te\n"
	cols := []string{"x", "y", "z"}
	rows, _ := parseTSV(raw, cols)
	if len(rows) != 1 || len(rows[0]) != 3 {
		t.Fatalf("want 1 row of 3 fields, got %d rows of %d fields", len(rows), len(rows[0]))
	}
}

func TestParseTSV_IgnoresBlankLines(t *testing.T) {
	raw := "\n2026-04-01\tStarbucks\t-5.75\n\n"
	cols := []string{"date", "description", "amount"}
	rows, _ := parseTSV(raw, cols)
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
}
