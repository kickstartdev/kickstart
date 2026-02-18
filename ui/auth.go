package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kickstartdev/kickstart/internal/auth"
)

type deviceCodeMsg struct {
	UserCode	string
	VerificationURI string
	DeviceCode		string
	Interval		int
}

// messages - sent back to update when background work is finished
type deviceCodeErrMsg struct {
	Err error
}

type tokenMsg struct {
	Token string
}

type tokenErrMsg struct {
	Err error
}

type usernameMsg struct {
	Username string
}

type authSuccessTimerMsg struct{}

func requestDeviceCodeCmd() tea.Msg {
	resp, err := auth.RequestDeviceCode()
	if err != nil {
		return deviceCodeErrMsg{Err: err}
	}
	return deviceCodeMsg{
		UserCode:	resp.UserCode,
		VerificationURI: resp.VerificationURI,
		DeviceCode: resp.DeviceCode,
		Interval: resp.Interval,
	}
}

func pollForTokenCmd(deviceCode string, interval int) tea.Cmd {
	return func() tea.Msg {
		token, err := auth.PollForToken(deviceCode, interval)
		if err != nil {
			return tokenErrMsg{Err: err}
		}
		return tokenMsg{Token: token}
	}
}

func getUsernameCmd(token string) tea.Cmd {
	return func() tea.Msg {
		username, err := auth.GetUsername(token)
		if err != nil {
			return usernameMsg{Username: "unknown"}
		}
		return usernameMsg{Username: username}
	}
}

func (m *Model) UpdateAuth(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
		case deviceCodeMsg:
			m.UserCode = msg.UserCode
			m.VerificationURI = msg.VerificationURI
			m.DeviceCode = msg.DeviceCode
			m.Interval = msg.Interval

			return m, pollForTokenCmd(msg.DeviceCode, msg.Interval)
		
		case deviceCodeErrMsg:
			m.AuthError = msg.Err.Error()
			return m, nil
		
		case tokenMsg:
			m.Token = msg.Token

			return m, getUsernameCmd(msg.Token)
		
		case tokenErrMsg:
			m.AuthError = msg.Err.Error()
			return m, nil
		
		case usernameMsg:
			m.Username = msg.Username
			auth.SaveConfig(auth.Config{
				Token: m.Token,
				Username: m.Username,
			})

			m.Screen = screenAuthSuccess
			m.UserCode = ""
			m.VerificationURI = ""
			m.DeviceCode = ""

			return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return authSuccessTimerMsg{}
			})

		case authSuccessTimerMsg:
			m.Screen = screenTemplates
			m.TemplatesLoading = true
			return m, m.fetchTemplateCmd
	}

	return m, nil

}


func (m *Model) ViewAuthSuccess() string {
	content := greenStyle.Render("Authenticated!") + "\n\n"
	content += dimStyle.Render("Welcome, ") + accentStyle.Render(m.Username)
	return m.Layout(content, "")
}

func (m *Model) ViewAuth() string {
	var content string
	if m.AuthError != "" {
		content = redStyle.Render("Error: "+m.AuthError) + "\n\n"
		content += dimStyle.Render("Press ") + accentStyle.Render("r") + dimStyle.Render(" to retry")
	} else if m.UserCode == "" {
		content = "Connecting to GitHub...  " + m.Spinner.View()
	} else {
		content = "1. Open this URL in your browser:\n"
		content += "   " + accentStyle.Render(m.VerificationURI) + "\n\n"
		content += "2. Enter this code:\n"
		content += "   " + accentStyle.Render(m.UserCode) + "\n\n"
		content += dimStyle.Render("Waiting for authorization...")
	}
	return m.Layout(content, "r retry    q quit")
}


