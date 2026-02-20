package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kickstartdev/kickstart/internal/github"
)

const (
	createStepMeta        = 0
	createStepVarName     = 1
	createStepVarDesc     = 2
	createStepVarDefault  = 3
	createStepVarRequired = 4
	createStepPublishing  = 5
	createStepDone        = 6
)

type createTemplatePublishedMsg struct {
	URL string
}

type createTemplateErrMsg struct {
	Err error
}

func (m *Model) initCreateTemplate() {
	m.CreateStep = createStepMeta
	m.CreateConfig = github.TemplateConfig{Branch: "main"}
	m.CreateVarCurrent = github.Variable{}
	m.CreatePublishErr = ""
	m.CreatePublishedURL = ""

	placeholders := []string{
		"e.g. Go REST API",
		"e.g. A scalable Go REST API starter",
		"e.g. go-rest-api-template",
	}
	m.CreateMetaInputs = make([]textinput.Model, 3)
	for i := range m.CreateMetaInputs {
		ti := textinput.New()
		ti.Placeholder = placeholders[i]
		ti.CharLimit = 100
		ti.Width = 40
		ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f0883e"))
		ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#e6edf3"))
		ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#30363d"))
		ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#f0883e"))
		m.CreateMetaInputs[i] = ti
	}
	m.CreateMetaInputs[0].Focus()
	m.CreateMetaCursor = 0

	vi := textinput.New()
	vi.CharLimit = 100
	vi.Width = 40
	vi.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f0883e"))
	vi.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#e6edf3"))
	vi.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#30363d"))
	vi.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#f0883e"))
	m.CreateVarInput = vi
}

func (m *Model) publishTemplateCmd() tea.Msg {
	url, err := github.CreateTemplate(github.NewTemplate{
		Token:    m.Token,
		Username: m.Username,
		RepoName: m.CreateRepoName,
		Config:   m.CreateConfig,
	})
	if err != nil {
		return createTemplateErrMsg{Err: err}
	}
	return createTemplatePublishedMsg{URL: url}
}

func (m *Model) UpdateCreateTemplate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case createTemplatePublishedMsg:
		m.CreatePublishedURL = msg.URL
		m.CreateStep = createStepDone
		return m, nil

	case createTemplateErrMsg:
		m.CreatePublishErr = msg.Err.Error()
		m.CreateStep = createStepDone
		return m, nil

	case tea.KeyMsg:
		// During publishing/done, only handle esc and q
		if m.CreateStep == createStepPublishing || m.CreateStep == createStepDone {
			switch msg.String() {
			case "esc":
				if m.CreatePublishErr != "" {
					m.CreateStep = createStepMeta
					m.CreatePublishErr = ""
					m.CreateMetaInputs[m.CreateMetaCursor].Focus()
				} else {
					m.Screen = screenTemplates
					m.TemplatesLoading = true
					return m, m.fetchTemplateCmd
				}
			case "q":
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.String() {
		case "esc":
			switch m.CreateStep {
			case createStepMeta:
				m.Screen = screenTemplates
				return m, nil
			case createStepVarName:
				m.CreateStep = createStepMeta
				m.CreateVarInput.Blur()
				m.CreateMetaInputs[m.CreateMetaCursor].Focus()
				return m, nil
			default:
				m.CreateVarCurrent = github.Variable{}
				m.CreateStep = createStepVarName
				m.CreateVarInput.SetValue("")
				m.CreateVarInput.Placeholder = "e.g. project_name"
				m.CreateVarInput.Focus()
				return m, nil
			}

		case "enter":
			return m.handleCreateEnter()

		case "tab", "down":
			if m.CreateStep == createStepMeta && m.CreateMetaCursor < len(m.CreateMetaInputs)-1 {
				m.CreateMetaInputs[m.CreateMetaCursor].Blur()
				m.CreateMetaCursor++
				m.CreateMetaInputs[m.CreateMetaCursor].Focus()
			}
			return m, nil

		case "shift+tab", "up":
			if m.CreateStep == createStepMeta && m.CreateMetaCursor > 0 {
				m.CreateMetaInputs[m.CreateMetaCursor].Blur()
				m.CreateMetaCursor--
				m.CreateMetaInputs[m.CreateMetaCursor].Focus()
			}
			return m, nil
		}
	}

	// Fall through: update active text input
	var cmd tea.Cmd
	if m.CreateStep == createStepMeta {
		m.CreateMetaInputs[m.CreateMetaCursor], cmd = m.CreateMetaInputs[m.CreateMetaCursor].Update(msg)
	} else if m.CreateStep >= createStepVarName && m.CreateStep <= createStepVarRequired {
		m.CreateVarInput, cmd = m.CreateVarInput.Update(msg)
	}
	return m, cmd
}

func (m *Model) handleCreateEnter() (tea.Model, tea.Cmd) {
	switch m.CreateStep {
	case createStepMeta:
		if m.CreateMetaCursor < len(m.CreateMetaInputs)-1 {
			m.CreateMetaInputs[m.CreateMetaCursor].Blur()
			m.CreateMetaCursor++
			m.CreateMetaInputs[m.CreateMetaCursor].Focus()
			return m, nil
		}
		m.CreateConfig.Name = m.CreateMetaInputs[0].Value()
		m.CreateConfig.Description = m.CreateMetaInputs[1].Value()
		m.CreateRepoName = m.CreateMetaInputs[2].Value()
		m.CreateMetaInputs[m.CreateMetaCursor].Blur()
		m.CreateStep = createStepVarName
		m.CreateVarInput.SetValue("")
		m.CreateVarInput.Placeholder = "e.g. project_name"
		m.CreateVarInput.Focus()
		return m, nil

	case createStepVarName:
		val := strings.TrimSpace(m.CreateVarInput.Value())
		if val == "" {
			m.CreateStep = createStepPublishing
			return m, m.publishTemplateCmd
		}
		m.CreateVarCurrent.Name = val
		m.CreateStep = createStepVarDesc
		m.CreateVarInput.SetValue("")
		m.CreateVarInput.Placeholder = "e.g. Name of your project"
		m.CreateVarInput.Focus()
		return m, nil

	case createStepVarDesc:
		m.CreateVarCurrent.Description = m.CreateVarInput.Value()
		m.CreateStep = createStepVarDefault
		m.CreateVarInput.SetValue("")
		m.CreateVarInput.Placeholder = "e.g. my-project"
		m.CreateVarInput.Focus()
		return m, nil

	case createStepVarDefault:
		m.CreateVarCurrent.Default = m.CreateVarInput.Value()
		m.CreateStep = createStepVarRequired
		m.CreateVarInput.SetValue("")
		m.CreateVarInput.Placeholder = "y / n"
		m.CreateVarInput.Focus()
		return m, nil

	case createStepVarRequired:
		val := strings.ToLower(strings.TrimSpace(m.CreateVarInput.Value()))
		m.CreateVarCurrent.Required = val == "y"
		m.CreateConfig.Variables = append(m.CreateConfig.Variables, m.CreateVarCurrent)
		m.CreateVarCurrent = github.Variable{}
		m.CreateStep = createStepVarName
		m.CreateVarInput.SetValue("")
		m.CreateVarInput.Placeholder = "e.g. author_name"
		m.CreateVarInput.Focus()
		return m, nil
	}

	return m, nil
}

func (m *Model) ViewCreateTemplate() string {
	switch m.CreateStep {
	case createStepMeta:
		return m.viewCreateMeta()
	case createStepVarName:
		return m.viewCreateVarName()
	case createStepVarDesc:
		return m.viewCreateVarField("Variable description", "What does this variable represent?")
	case createStepVarDefault:
		return m.viewCreateVarField("Default value", "Leave empty for no default")
	case createStepVarRequired:
		return m.viewCreateVarField("Required? (y/n)", "")
	case createStepPublishing:
		return m.Layout("Publishing your template...  "+m.Spinner.View(), "")
	case createStepDone:
		return m.viewCreateDone()
	}
	return ""
}

func (m *Model) viewCreateMeta() string {
	labels := []string{"Template name", "Description", "Repo name"}
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#e6edf3"))
	rowStyle := lipgloss.NewStyle().Width(22)

	s := "Create a new template:\n\n"
	for i, label := range labels {
		cursor := "  "
		if m.CreateMetaCursor == i {
			cursor = accentStyle.Render(") ")
		}
		row := rowStyle.Render(cursor + labelStyle.Render(label))
		s += row + m.CreateMetaInputs[i].View() + "\n"
		if m.CreateMetaCursor == i {
			s += "\n"
		}
	}

	return m.Layout(s, "tab next   shift+tab back   enter confirm   esc cancel")
}

func (m *Model) viewCreateVarName() string {
	s := accentStyle.Render(m.CreateConfig.Name) + "\n\n"

	if len(m.CreateConfig.Variables) > 0 {
		s += "Variables added:\n"
		for _, v := range m.CreateConfig.Variables {
			req := ""
			if v.Required {
				req = dimStyle.Render("  required")
			}
			s += greenStyle.Render("  + {{"+v.Name+"}}") + req + "\n"
		}
		s += "\n"
	}

	s += "Add a variable " + dimStyle.Render("(leave name empty to finish)") + "\n\n"
	s += m.CreateVarInput.View()

	return m.Layout(s, "enter next   esc back")
}

func (m *Model) viewCreateVarField(label, hint string) string {
	s := accentStyle.Render(m.CreateConfig.Name) + "\n\n"
	s += dimStyle.Render("Adding: ") + accentStyle.Render("{{"+m.CreateVarCurrent.Name+"}}") + "\n\n"
	s += label + "\n\n"
	s += m.CreateVarInput.View()
	if hint != "" {
		s += "\n\n" + dimStyle.Render(hint)
	}
	return m.Layout(s, "enter next   esc cancel variable")
}

func (m *Model) viewCreateDone() string {
	if m.CreatePublishErr != "" {
		s := redStyle.Render("Failed to publish template") + "\n\n"
		s += dimStyle.Render(m.CreatePublishErr) + "\n\n"
		s += "Press " + accentStyle.Render("esc") + " to go back"
		return m.Layout(s, "esc back   q quit")
	}

	s := greenStyle.Render("Template published!") + "\n\n"
	s += "  " + dimStyle.Render("Name:  ") + accentStyle.Render(m.CreateConfig.Name) + "\n"
	s += "  " + dimStyle.Render("Repo:  ") + accentStyle.Render(m.CreatePublishedURL) + "\n\n"
	s += dimStyle.Render("It will now appear in the kickstart template list.\n\n")
	s += "Press " + accentStyle.Render("esc") + dimStyle.Render(" to return to templates")
	return m.Layout(s, "esc templates   q quit")
}
