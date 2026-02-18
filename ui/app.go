package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kickstartdev/kickstart/internal/auth"
	"github.com/kickstartdev/kickstart/internal/debug"
	"github.com/kickstartdev/kickstart/internal/github"
	"github.com/kickstartdev/kickstart/internal/scaffold"
)

type Model struct {
	Screen   string
	Choices  []string
	Cursor   int
	Selected map[int]struct{}
	Width    int
	Height   int

	// auth
	UserCode        string
	VerificationURI string
	Interval        int
	DeviceCode      string
	Token           string
	Username        string
	AuthError       string

	// spinner
	Spinner spinner.Model


	//table
	Templates []github.Template
	TemplatesLoading bool
	TemplatesError string
	SelectedTemplate github.Template
	Table	table.Model

	//form
	FormInputs []textinput.Model
	FormCursor	int
	FormValues map[string]string
	FormLoading bool
	FormError	string

	//scaffolding
	Scaffolder      *scaffold.Scaffolder
	ScaffoldSteps   []scaffoldStep
	ScaffoldCurrent int
	ScaffoldError   string



}

const (
	screenWelcome     = "welcome"
	screenAuth        = "auth"
	screenAuthSuccess = "auth_success"
	screenTemplates   = "templates"
	screenForm        = "form"
	screenScaffolding = "scaffolding"
	screenSuccess     = "success"
)

func NewApp() *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	cfg, err := auth.LoadConfig()

	if err == nil && cfg.Token != "" {
		debug.Log("NewApp: found token for user %s, going to templates", cfg.Username)
		return &Model{
			Screen:           screenTemplates,
			Token:            cfg.Token,
			Username:         cfg.Username,
			Spinner:          s,
			TemplatesLoading: true,
		}
	}

	return &Model{
		Screen:  screenWelcome,
		Spinner: s,
	}
}

func (m *Model) Init() tea.Cmd {
	debug.Log("Init: screen=%s templatesLoading=%v", m.Screen, m.TemplatesLoading)
	if m.Screen == screenTemplates {
		return tea.Batch(m.Spinner.Tick, m.fetchTemplateCmd)
	}
	return m.Spinner.Tick
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case	"q":
			if m.Screen != screenForm {
				return m, tea.Quit
			}
		case "enter":
			if m.Screen == screenWelcome {
				m.Screen = screenAuth
				return m, requestDeviceCodeCmd
			}
		case "r":
			if m.Screen == screenAuth && m.AuthError != "" {
				m.AuthError = ""
				return m, requestDeviceCodeCmd
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
		
	}

	switch m.Screen {
	case screenAuth, screenAuthSuccess:
		return m.UpdateAuth(msg)
	case screenTemplates:
		return m.UpdateTemplates(msg)
	case screenForm:
		return m.UpdateForm(msg)
	case screenScaffolding:
		return m.UpdateScaffolding(msg)
	}
	

	return m, nil
}

func (m *Model) View() string {
	switch m.Screen {
	case screenWelcome:
		return m.ViewWelcome()

	case screenAuth:
		return m.ViewAuth()
	case screenAuthSuccess:
		return m.ViewAuthSuccess()
	case screenTemplates:
		return m.ViewTemplates()
	case screenForm:
		return m.ViewForm()
	case screenScaffolding:
		return m.ViewScaffolding()
	case screenSuccess:
		return m.ViewSuccess()
	default:
		return ""
	}
}
