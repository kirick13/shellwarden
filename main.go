package main

import (
	"log"
	"time"

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
	Hosts []bw.Host
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

type bwGetHostsSuccessMsg struct {
	Hosts []bw.Host
}

type bwGetHostsErrorMsg struct {
	Err error
}

type bwBootstrapSuccessMsg struct {
	Hosts       []bw.Host
	NeedsUnlock bool
}

type bwBootstrapErrorMsg struct {
	Err error
}

type hostsPollTickMsg struct{}
type unlockPollTickMsg struct{}

func initialModel() model {
	return model{
		display: display.NewDisplay(),
	}
}

func (m model) Init() tea.Cmd {
	if spinnerView, ok := m.display.CurrentView.(*view.SpinnerView); ok {
		return tea.Batch(spinnerView.Init(), bootstrapCmd())
	}
	if _, ok := m.display.CurrentView.(*view.HostsView); ok {
		return tea.Batch(bootstrapCmd(), hostsPollTickCmd())
	}
	return bootstrapCmd()
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

	case bwBootstrapSuccessMsg:
		if msg.NeedsUnlock {
			m.display.SetCurrentView(view.NewBwUnlockView())
			return m, tea.Batch(viewCmd, unlockPollTickCmd())
		}
		m.display.SetCurrentView(view.NewHostsView(msg.Hosts))
		return m, tea.Batch(viewCmd, hostsPollTickCmd())

	case bwBootstrapErrorMsg:
		m.display.SetCurrentView(view.NewTextView(msg.Err.Error()))
		return m, viewCmd

	case bwUnlockSuccessMsg:
		m.display.SetCurrentView(view.NewHostsView(msg.Hosts))
		return m, tea.Batch(viewCmd, hostsPollTickCmd())

	case bwUnlockErrorMsg:
		unlockView := view.NewBwUnlockView()
		if msg.Err != bw.ErrServerExists {
			unlockView.SetError(msg.Err.Error())
		}
		m.display.SetCurrentView(unlockView)
		return m, tea.Batch(viewCmd, unlockPollTickCmd())

	case bwReloadSuccessMsg:
		selectedID := ""
		if hostsView, ok := m.display.CurrentView.(*view.HostsView); ok {
			selectedID = hostsView.SelectedHostID()
		}
		m.display.SetCurrentView(view.NewHostsViewWithSelection(msg.Hosts, selectedID))
		return m, viewCmd

	case bwReloadErrorMsg:
		spinnerView := view.NewSpinnerView("starting Shellwarden...")
		m.display.SetCurrentView(spinnerView)
		return m, tea.Batch(spinnerView.Init(), bootstrapCmd())

	case bwGetHostsSuccessMsg:
		if hostsView, ok := m.display.CurrentView.(*view.HostsView); ok {
			m.display.SetCurrentView(view.NewHostsViewWithSelection(msg.Hosts, hostsView.SelectedHostID()))
			return m, viewCmd
		}
		if _, ok := m.display.CurrentView.(*view.BwUnlockView); ok {
			m.display.SetCurrentView(view.NewHostsView(msg.Hosts))
			return m, tea.Batch(viewCmd, hostsPollTickCmd())
		}
		return m, viewCmd

	case bwGetHostsErrorMsg:
		if _, ok := m.display.CurrentView.(*view.HostsView); ok {
			spinnerView := view.NewSpinnerView("starting Shellwarden...")
			m.display.SetCurrentView(spinnerView)
			return m, tea.Batch(spinnerView.Init(), bootstrapCmd())
		}
		return m, viewCmd

	case hostsPollTickMsg:
		if _, ok := m.display.CurrentView.(*view.HostsView); ok {
			return m, tea.Batch(getHostsCmd(), hostsPollTickCmd())
		}
		return m, viewCmd

	case unlockPollTickMsg:
		if _, ok := m.display.CurrentView.(*view.BwUnlockView); ok {
			return m, tea.Batch(getHostsCmd(), unlockPollTickCmd())
		}
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

				spinnerView := view.NewSpinnerView("unlocking Bitwarden...")
				m.display.SetCurrentView(spinnerView)
				return m, tea.Batch(spinnerView.Init(), unlockHostsCmd(password))
			}

			m.display.CurrentView.OnEnter()
		case "r":
			if _, ok := m.display.CurrentView.(*view.HostsView); ok {
				spinnerView := view.NewSpinnerView("updating hosts...")
				m.display.SetCurrentView(spinnerView)
				return m, tea.Batch(spinnerView.Init(), reloadHostsCmd())
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

func bootstrapCmd() tea.Cmd {
	return func() tea.Msg {
		result, err := bw.Bootstrap()
		if err != nil {
			return bwBootstrapErrorMsg{Err: err}
		}

		return bwBootstrapSuccessMsg{
			Hosts:       result.Hosts,
			NeedsUnlock: result.NeedsUnlock,
		}
	}
}

func unlockHostsCmd(password string) tea.Cmd {
	return func() tea.Msg {
		hosts, err := bw.UnlockHosts(password)
		if err != nil {
			return bwUnlockErrorMsg{Err: err}
		}

		return bwUnlockSuccessMsg{
			Hosts: hosts,
		}
	}
}

func reloadHostsCmd() tea.Cmd {
	return func() tea.Msg {
		hosts, err := bw.ReloadHosts()
		if err != nil {
			return bwReloadErrorMsg{Err: err}
		}

		return bwReloadSuccessMsg{
			Hosts: hosts,
		}
	}
}

func hostsPollTickCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return hostsPollTickMsg{}
	})
}

func unlockPollTickCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(time.Time) tea.Msg {
		return unlockPollTickMsg{}
	})
}

func getHostsCmd() tea.Cmd {
	return func() tea.Msg {
		hosts, err := bw.GetHosts()
		if err != nil {
			return bwGetHostsErrorMsg{Err: err}
		}

		return bwGetHostsSuccessMsg{
			Hosts: hosts,
		}
	}
}
