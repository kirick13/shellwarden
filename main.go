package main

import (
	"log"

	tea "charm.land/bubbletea/v2"
	"github.com/kirick13/shellwarden/bw"
	display "github.com/kirick13/shellwarden/components"
	view "github.com/kirick13/shellwarden/components/view"
)

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type model struct {
	display  *display.Display
	quitting bool
}

type bwUnlockSuccessMsg struct {
	Session string
	Hosts   []bw.Host
}

type bwUnlockErrorMsg struct {
	Err error
}

type bwReloadSuccessMsg struct {
	Hosts []bw.Host
}

type bwReloadErrorMsg struct {
	Err error
}

func initialModel() model {
	return model{
		display: display.NewDisplay(),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var viewCmd tea.Cmd
	if m.display != nil && m.display.CurrentView != nil {
		viewCmd = m.display.CurrentView.OnMsg(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.display.UpdateDocumentSize(msg.Width, msg.Height)
		return m, viewCmd

	case bwUnlockSuccessMsg:
		m.display.SetBWSession(msg.Session)
		m.display.SetCurrentView(view.NewHostsView(msg.Hosts))
		return m, viewCmd

	case bwUnlockErrorMsg:
		unlockView := view.NewBwUnlockView()
		unlockView.SetError(msg.Err.Error())
		m.display.SetCurrentView(unlockView)
		return m, viewCmd

	case bwReloadSuccessMsg:
		m.display.SetCurrentView(view.NewHostsView(msg.Hosts))
		return m, viewCmd

	case bwReloadErrorMsg:
		m.display.SetCurrentView(view.NewTextView(msg.Err.Error()))
		return m, viewCmd

	// case sshFinishedMsg:
	// 	return m, tea.ClearScreen

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "tab":
			m.display.MoveFocus(1)
			return m, nil
		case "shift+tab":
			m.display.MoveFocus(-1)
			return m, nil
		case "enter":
			if unlockView, ok := m.display.CurrentView.(*view.BwUnlockView); ok {
				password := unlockView.Password()
				if password == "" {
					unlockView.SetError("Password is required.")
					return m, viewCmd
				}

				progressView := view.NewSpinnerView("unlocking Bitwarden...")
				m.display.SetCurrentView(progressView)
				return m, tea.Batch(progressView.Init(), unlockBWCmd(password))
			}

			m.display.CurrentView.OnEnter()
		case "r":
			if _, ok := m.display.CurrentView.(*view.HostsView); ok {
				session := m.display.BWSession()
				if session == "" {
					return m, viewCmd
				}

				spinnerView := view.NewSpinnerView("updating hosts...")
				m.display.SetCurrentView(spinnerView)
				return m, tea.Batch(spinnerView.Init(), reloadHostsCmd(session))
			}

			m.display.CurrentView.OnKey(msg.String())
		case "esc":
			m.display.CurrentView.OnEsc()
		default:
			m.display.CurrentView.OnKey(msg.String())

			// if missing := m.validate(); missing != "" {
			// 	return m, nil
			// }

			// return m, tea.ExecProcess(m.sshCommand(), func(error) tea.Msg {
			// 	return sshFinishedMsg{}
			// })
		}
	}

	if m.display == nil || m.display.CurrentView == nil {
		return m, viewCmd
	}

	inputs := m.display.CurrentView.Inputs()
	cmds := make([]tea.Cmd, len(inputs))
	for i := range inputs {
		inputs[i], cmds[i] = inputs[i].Update(msg)
	}

	cmds = append(cmds, viewCmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	// str := lipgloss.JoinVertical(lipgloss.Top, m.inputs[0].Render(), "", m.inputs[1].Render(), "", m.inputs[2].Render())

	// layers := []*lipgloss.Layer{
	// 	// lipgloss.NewLayer(bg),
	// 	lipgloss.NewLayer(str),
	// }

	// comp := lipgloss.NewCompositor(layers...)
	// comp.Render()

	return tea.NewView(m.display.Render())
}

func unlockBWCmd(password string) tea.Cmd {
	return func() tea.Msg {
		result, err := bw.Unlock(password)
		if err != nil {
			return bwUnlockErrorMsg{Err: err}
		}

		return bwUnlockSuccessMsg{
			Session: result.Session,
			Hosts:   result.Hosts,
		}
	}
}

func reloadHostsCmd(session string) tea.Cmd {
	return func() tea.Msg {
		hosts, err := bw.ListHosts(session)
		if err != nil {
			return bwReloadErrorMsg{Err: err}
		}

		return bwReloadSuccessMsg{
			Hosts: hosts,
		}
	}
}
