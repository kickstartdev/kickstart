package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kickstartdev/kickstart/internal/github"
)

type templateConfigLoadedMsg struct {
	Config github.TemplateConfig
}

type templateConfigErrMsg struct {
	Err error
}

func (m *Model) fetchTemplateLoadedConfigCmd() tea.Msg {
	cfg, err := github.GetTemplateConfig(
		m.Token,
		m.SelectedTemplate.Owner,
		m.SelectedTemplate.Repo,
	)

	if err != nil {
		return templateConfigErrMsg{Err: err}
	}

	return templateConfigLoadedMsg{Config: *cfg}
}

func (m *Model) buildFormInputs() {
	m.FormInputs = make([]textinput.Model, len(m.SelectedTemplate.Config.Variables))

	for i := range m.SelectedTemplate.Config.Variables {
		ti := textinput.New()
		ti.Placeholder = ""
		ti.CharLimit = 100
		ti.Width = 40
		ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f0883e"))
		ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#e6edf3"))
		ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#30363d"))
		ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#f0883e"))

		if i == 0 {
			ti.Focus()
		}

		m.FormInputs[i] = ti
	}

	m.FormCursor = 0

}

func (m *Model) UpdateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case templateConfigLoadedMsg:
		m.SelectedTemplate.Config = msg.Config
		m.FormLoading = false
		m.buildFormInputs()
		return m, nil
	
	case templateConfigErrMsg:
		m.FormError	= msg.Err.Error()
		m.FormLoading = false
		return m,nil

	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			if m.FormCursor < len(m.FormInputs)-1 {
				m.FormInputs[m.FormCursor].Blur()
				m.FormCursor++
				m.FormInputs[m.FormCursor].Focus()
			}

			return m, nil
		
		case "shift+tab", "up":
			if m.FormCursor > 0 {
				m.FormInputs[m.FormCursor].Blur()
				m.FormCursor--
				m.FormInputs[m.FormCursor].Focus()
			}

			return m, nil
		
		case "enter":
			if m.FormCursor == len(m.FormInputs)-1 {
				m.collectFormValues()
				m.Screen = screenScaffolding
				return m, m.startScaffoldingCmd
			}

			m.FormInputs[m.FormCursor].Blur()
			m.FormCursor++
			m.FormInputs[m.FormCursor].Focus()
			return m,nil

		case "esc" :
			m.Screen = screenTemplates
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.FormInputs[m.FormCursor], cmd = m.FormInputs[m.FormCursor].Update(msg)
	return m, cmd

	
}

func (m *Model) collectFormValues() {
	m.FormValues = make(map[string]string)
	for i, v := range m.SelectedTemplate.Config.Variables {
		value := m.FormInputs[i].Value()
		if value == "" {
			value = v.Default
		}
		m.FormValues[v.Name] = value
	}
}



func (m *Model) ViewForm() string {
	if m.FormLoading {
		return m.Layout("Loading template config .... "+m.Spinner.View(), "q quit")
	}

	if m.FormError != "" {
		content := redStyle.Render("Error: "+m.FormError) + "\n\n"
		content += "Press " + accentStyle.Render("esc") + " to go back"
		return m.Layout(content, "esc back   q quit")
	}

	template := m.SelectedTemplate.Config
	s := accentStyle.Render(template.Name) + "  " + dimStyle.Render(template.Description) + "\n"
	s += dimStyle.Render(m.SelectedTemplate.Owner+"/"+m.SelectedTemplate.Repo) + "\n\n"
	s += "Configure your project:\n\n"

	labelColor := lipgloss.NewStyle().Foreground(lipgloss.Color("#e6edf3"))
	requiredStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f85149"))
	rowStyle := lipgloss.NewStyle().Width(22)

	for i, v := range template.Variables {
		cursor := "  "
		if m.FormCursor == i {
			cursor = accentStyle.Render(") ")
		}

		label := labelColor.Render(v.Name)
		if v.Required {
			label += requiredStyle.Render(" *")
		}

		row := rowStyle.Render(cursor + label)
		s += row + m.FormInputs[i].View() + "\n"

		if m.FormCursor == i {
			hint := v.Description
			if v.Default != "" {
				if hint != "" {
					hint += " "
				}
				hint += "(default: " + v.Default + ")"
			}
			if hint != "" {
				s += lipgloss.NewStyle().PaddingLeft(22).Render(dimStyle.Render(hint)) + "\n"
			}
		}
	}

	return m.Layout(s, "tab next   shift+tab back   enter submit   esc cancel")
}