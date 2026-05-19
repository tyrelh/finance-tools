package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	pickerScreen screen = iota
	formScreen
	runningScreen
	outputScreen
	errorScreen
)

type scriptItem struct {
	s Script
}

func (i scriptItem) Title() string       { return i.s.Name }
func (i scriptItem) Description() string { return i.s.Description }
func (i scriptItem) FilterValue() string { return i.s.Name }

type model struct {
	screen        screen
	width, height int

	list    list.Model
	spinner spinner.Model
	table   table.Model

	inv *Invocation

	tsv         string
	stderr      string
	copyNotice  string
	runErr      error
	fatalErr    error

	authChecking bool
	authOk       bool
	authMessage  string
}

func initialModel() model {
	items := make([]list.Item, len(registry))
	for i, s := range registry {
		items[i] = scriptItem{s: s}
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "Wealthsimple Finances"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(len(registry) > 5)
	l.Styles.Title = titleStyle

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		screen:       pickerScreen,
		list:         l,
		spinner:      sp,
		authChecking: true,
		authMessage:  "Checking session…",
	}
}

func (m model) Init() tea.Cmd {
	return runAuthCheck()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width, msg.Height-3)
		if m.inv != nil && m.inv.Form != nil {
			m.inv.Form = m.inv.Form.WithWidth(msg.Width).WithHeight(msg.Height - 2)
		}
		if len(m.table.Columns()) > 0 {
			m.table.SetHeight(maxInt(5, msg.Height-6))
		}
		return m, nil

	case tea.KeyMsg:
		switch m.screen {
		case pickerScreen:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "enter":
				if it, ok := m.list.SelectedItem().(scriptItem); ok {
					m.inv = it.s.New()
					if m.width > 0 {
						m.inv.Form = m.inv.Form.WithWidth(m.width).WithHeight(m.height - 2)
					}
					m.screen = formScreen
					return m, m.inv.Form.Init()
				}
			}
		case outputScreen:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "esc":
				return m.resetToPicker(), nil
			case "c":
				if err := clipboard.WriteAll(m.tsv); err != nil {
					m.copyNotice = errorStyle.Render("copy failed: " + err.Error())
				} else {
					m.copyNotice = successStyle.Render("✓ copied TSV to clipboard")
				}
				return m, nil
			}
		case errorScreen:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "esc", "enter":
				return m.resetToPicker(), nil
			}
		case runningScreen:
			// ignore key input while running
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
		}

	case authCheckMsg:
		m.authChecking = false
		m.authOk = msg.Ok
		m.authMessage = msg.Message
		return m, nil

	case runFinishedMsg:
		m.stderr = msg.Stderr
		m.tsv = msg.Stdout
		if msg.Err != nil {
			m.runErr = msg.Err
			m.screen = errorScreen
			return m, nil
		}
		var followup tea.Cmd
		if m.inv != nil && m.inv.ScriptFile == "ws_auth.py" {
			m.authChecking = true
			m.authMessage = "Checking session…"
			followup = runAuthCheck()
		}
		columns := m.inv.Columns()
		m.copyNotice = ""
		if len(columns) == 0 {
			m.table = table.Model{}
			m.screen = outputScreen
			return m, followup
		}
		rows, tableCols := parseTSV(msg.Stdout, columns)
		t := table.New(
			table.WithColumns(tableCols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(maxInt(5, m.height-6)),
		)
		ts := table.DefaultStyles()
		ts.Header = ts.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).BorderBottom(true).Bold(true)
		ts.Selected = ts.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
		t.SetStyles(ts)
		m.table = t
		m.screen = outputScreen
		return m, followup
	}

	// Route to the active screen's component.
	switch m.screen {
	case pickerScreen:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd

	case formScreen:
		form, cmd := m.inv.Form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.inv.Form = f
		}
		switch m.inv.Form.State {
		case huh.StateCompleted:
			args := m.inv.BuildArgs()
			var stdin string
			if m.inv.Stdin != nil {
				stdin = m.inv.Stdin()
			}
			m.screen = runningScreen
			return m, tea.Batch(m.spinner.Tick, runScript(m.inv.ScriptFile, args, stdin))
		case huh.StateAborted:
			return m.resetToPicker(), nil
		}
		return m, cmd

	case runningScreen:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case outputScreen:
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	switch m.screen {
	case pickerScreen:
		return m.list.View() + "\n" + m.authStatusLine()

	case formScreen:
		return m.inv.Form.View()

	case runningScreen:
		return fmt.Sprintf("\n  %s Running script…\n\n%s",
			m.spinner.View(),
			footerStyle.Render("ctrl+c to abort"))

	case outputScreen:
		var b strings.Builder
		b.WriteString(titleStyle.Render(fmt.Sprintf(" %s ", m.inv.ScriptFile)))
		b.WriteString("\n")
		hasTable := len(m.inv.Columns()) > 0
		if hasTable {
			b.WriteString(tableBorderStyle.Render(m.table.View()))
			b.WriteString("\n")
			rowCount := len(m.table.Rows())
			b.WriteString(subtleStyle.Render(fmt.Sprintf("%d row(s)", rowCount)))
			if m.copyNotice != "" {
				b.WriteString("   ")
				b.WriteString(m.copyNotice)
			}
		} else if out := strings.TrimSpace(m.tsv); out != "" {
			b.WriteString(out)
		}
		if m.stderr != "" {
			b.WriteString("\n")
			b.WriteString(stderrStyle.Render(strings.TrimSpace(m.stderr)))
		}
		b.WriteString("\n")
		if hasTable {
			b.WriteString(footerStyle.Render("↑/↓ navigate · c copy TSV · esc back · q quit"))
		} else {
			b.WriteString(footerStyle.Render("esc back · q quit"))
		}
		return b.String()

	case errorScreen:
		var b strings.Builder
		b.WriteString(errorStyle.Render("Script failed"))
		b.WriteString("\n\n")
		if m.runErr != nil {
			b.WriteString(m.runErr.Error())
			b.WriteString("\n")
		}
		if m.stderr != "" {
			b.WriteString("\n")
			b.WriteString(stderrStyle.Render(strings.TrimSpace(m.stderr)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(footerStyle.Render("esc/enter back · q quit"))
		return b.String()
	}

	return ""
}

func (m model) authStatusLine() string {
	if m.authChecking {
		return subtleStyle.Render("⋯ " + m.authMessage)
	}
	if m.authMessage == "" {
		return ""
	}
	if m.authOk {
		return successStyle.Render("✓ " + m.authMessage)
	}
	return errorStyle.Render("✗ " + m.authMessage)
}

func (m model) resetToPicker() model {
	m.screen = pickerScreen
	m.inv = nil
	m.tsv = ""
	m.stderr = ""
	m.copyNotice = ""
	m.runErr = nil
	return m
}

func parseTSV(raw string, columns []string) ([]table.Row, []table.Column) {
	widths := make([]int, len(columns))
	for i, c := range columns {
		widths[i] = len(c)
	}

	var rows []table.Row
	for _, line := range strings.Split(strings.TrimRight(raw, "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if rowMatchesHeader(fields, columns) {
			continue
		}
		for len(fields) < len(columns) {
			fields = append(fields, "")
		}
		if len(fields) > len(columns) {
			fields = fields[:len(columns)]
		}
		for i, f := range fields {
			if l := lipgloss.Width(f); l > widths[i] {
				widths[i] = l
			}
		}
		rows = append(rows, table.Row(fields))
	}

	cols := make([]table.Column, len(columns))
	for i, c := range columns {
		w := widths[i] + 2
		if w > 60 {
			w = 60
		}
		cols[i] = table.Column{Title: c, Width: w}
	}
	return rows, cols
}

func rowMatchesHeader(fields, columns []string) bool {
	if len(fields) != len(columns) {
		return false
	}
	for i := range fields {
		if fields[i] != columns[i] {
			return false
		}
	}
	return true
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	if _, err := tea.NewProgram(initialModel(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
