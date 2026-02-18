package ui

import (
	"fmt"

	"github.com/kickstartdev/kickstart/internal/scaffold"
	tea "github.com/charmbracelet/bubbletea"
)

type scaffoldStepDoneMsg struct {
	StepIndex int
}

type scaffoldErrMsg struct {
	Err error
}

type scaffoldCompleteMsg struct{}

func (m *Model) startScaffoldingCmd() tea.Msg {
	branch := m.SelectedTemplate.Config.Branch
	if branch == "" {
		branch = "main"
	}

	m.Scaffolder = scaffold.New(
		m.Token,
		m.SelectedTemplate.Owner,
		m.SelectedTemplate.Repo,
		branch,
		m.FormValues["project_name"],
		m.FormValues,
	)

	steps := m.Scaffolder.Steps()
	m.ScaffoldSteps = make([]scaffoldStep, len(steps))
	for i, s := range steps {
		m.ScaffoldSteps[i] = scaffoldStep{Name: s.Name, Status: "pending"}
	}
	m.ScaffoldSteps[0].Status = "running"
	m.ScaffoldCurrent = 0

	// run first step
	err := steps[0].Fn()
	if err != nil {
		return scaffoldErrMsg{Err: err}
	}
	return scaffoldStepDoneMsg{StepIndex: 0}
}

func (m *Model) runScaffoldStepCmd(stepIndex int) tea.Cmd {
	return func() tea.Msg {
		steps := m.Scaffolder.Steps()

		if stepIndex >= len(steps) {
			return scaffoldCompleteMsg{}
		}

		err := steps[stepIndex].Fn()
		if err != nil {
			return scaffoldErrMsg{Err: err}
		}
		return scaffoldStepDoneMsg{StepIndex: stepIndex}
	}
}

type scaffoldStep struct {
	Name   string
	Status string
}

func (m *Model) UpdateScaffolding(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case scaffoldStepDoneMsg:
		m.ScaffoldSteps[msg.StepIndex].Status = "done"
		next := msg.StepIndex + 1
		if next < len(m.ScaffoldSteps) {
			m.ScaffoldCurrent = next
			m.ScaffoldSteps[next].Status = "running"
			return m, m.runScaffoldStepCmd(next)
		}
		m.Screen = screenSuccess
		return m, nil

	case scaffoldErrMsg:
		m.ScaffoldSteps[m.ScaffoldCurrent].Status = "error"
		m.ScaffoldError = msg.Err.Error()
		return m, nil
	}

	return m, nil
}

func (m *Model) ViewSuccess() string {
	projectName := m.FormValues["project_name"]

	content := greenStyle.Render("Project scaffolded successfully!") + "\n\n"
	content += "  " + dimStyle.Render("Project:") + "   " + accentStyle.Render(projectName) + "\n"
	content += "  " + dimStyle.Render("Location:") + "  " + accentStyle.Render("./"+projectName) + "\n\n"
	content += "  " + dimStyle.Render("cd ") + accentStyle.Render(projectName) + dimStyle.Render(" to get started")

	return m.Layout(content, "q quit")
}

func (m *Model) ViewScaffolding() string {
	projectName := m.FormValues["project_name"]
	s := "Scaffolding " + accentStyle.Render(projectName) + "...\n\n"

	for _, step := range m.ScaffoldSteps {
		switch step.Status {
		case "done":
			s += fmt.Sprintf("  %s  %s\n", greenStyle.Render("✓"), dimStyle.Render(step.Name))
		case "running":
			s += fmt.Sprintf("  %s  %s\n", m.Spinner.View(), step.Name)
		case "error":
			s += fmt.Sprintf("  %s  %s\n", redStyle.Render("✗"), redStyle.Render(step.Name))
		default:
			s += fmt.Sprintf("  %s  %s\n", dimStyle.Render("○"), dimStyle.Render(step.Name))
		}
	}

	if m.ScaffoldError != "" {
		s += "\n" + redStyle.Render("Error: "+m.ScaffoldError)
	}

	s += fmt.Sprintf("\n\n%s", dimStyle.Render(fmt.Sprintf("step %d of %d", m.ScaffoldCurrent+1, len(m.ScaffoldSteps))))

	return m.Layout(s, "q quit")
}
