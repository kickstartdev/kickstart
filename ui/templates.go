package ui

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kickstartdev/kickstart/internal/debug"
	"github.com/kickstartdev/kickstart/internal/github"
)

type templatesLoadedMsg struct {
	Templates []github.Template
}


type templatesErrMsg struct {
	Err error
}

func (m *Model) fetchTemplateCmd() tea.Msg {
	debug.Log("fetchTemplateCmd: fetching templates for user=%s", m.Username)
	templates, err := github.ListTemplates(m.Token, m.Username)
	if err != nil {
		debug.Log("fetchTemplateCmd: error: %v", err)
		return templatesErrMsg{Err: err}
	}
	debug.Log("fetchTemplateCmd: loaded %d templates", len(templates))
	return templatesLoadedMsg{Templates: templates}
}


func (m *Model) buildTable() {
	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Description", Width: 50},
		{Title: "Source", Width: 30},
	}
	rows := []table.Row{}
	for _, t := range m.Templates {
		rows = append(rows, table.Row{
			t.Config.Name,
			t.Config.Description,
			t.Owner + "/" + t.Repo,
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithWidth(110),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("#30363d")).
	BorderBottom(true).
	Bold(true).
	Foreground(lipgloss.Color("#f0883e"))

	s.Selected = s.Selected.
	Foreground(lipgloss.Color("#e6edf3")).
	Background(lipgloss.Color("#f0883e")).
	Bold(true)

	s.Cell = s.Cell.Foreground(lipgloss.Color("#7d8590"))

	t.SetStyles(s)

	m.Table = t
}


func (m *Model) UpdateTemplates(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case templatesLoadedMsg:
		debug.Log("UpdateTemplates: received %d templates", len(msg.Templates))
		m.Templates = msg.Templates
		m.TemplatesLoading = false
		m.buildTable()
		return m, tea.ClearScreen

	case templatesErrMsg:
		debug.Log("UpdateTemplates: error: %v", msg.Err)
		m.TemplatesError = msg.Err.Error()
		m.TemplatesLoading = false
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter" :
			if len(m.Templates) > 0 {
				idx := m.Table.Cursor()
				m.SelectedTemplate = m.Templates[idx]
				m.Screen = screenForm
				m.FormLoading = true
				return m, m.fetchTemplateLoadedConfigCmd
			}
		
		case "r":
			if m.TemplatesError != "" {
				m.TemplatesError = ""
				m.TemplatesLoading = true
				return m, m.fetchTemplateCmd
			}
		}	
	}


	var cmd tea.Cmd
	m.Table, cmd = m.Table.Update(msg)
	return m, cmd
}

func (m *Model) ViewTemplates() string {
	if m.TemplatesLoading {
		return m.Layout("Searching your repo for templates...	"+dimStyle.Render("⠸"), "q quit")	
	}

	if m.TemplatesError != "" {
		content := redStyle.Render("Error: "+m.TemplatesError) + "\n\n"
		content += "Press	"+accentStyle.Render("r") + "	to retry"
		return m.Layout(content, "r retry		q quit")
	}

	if len(m.Templates) == 0 {
		content := dimStyle.Render("No templates found (0)") + "\n\n"
		content += "Add a " + accentStyle.Render("template.yaml") + " to a repo to get started"
		return m.Layout(content, "r refresh    q quit")
	}

	content := "Select a template:\n\n"
	content += m.Table.View()

	return m.Layout(content, "↑/↓ navigate    enter select    q quit")
}