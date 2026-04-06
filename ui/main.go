package main

import (
	"log"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/kirick13/shellwarden/shared"
	"github.com/kirick13/shellwarden/ui/components"
	view "github.com/kirick13/shellwarden/ui/components/view"
)

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type model struct {
	display           *components.Display
	quitting          bool
	clearOnNextRender bool
}

type bwUnlockSuccessMsg struct {
	Hosts []shared.Host
}

type bwUnlockErrorMsg struct {
	Err error
}

type bwReloadSuccessMsg struct {
	Hosts []shared.Host
}

type bwReloadErrorMsg struct {
	Err error
}

type bwGetHostsSuccessMsg struct {
	Hosts []shared.Host
}

type bwGetHostsErrorMsg struct {
	Err error
}

type bwBootstrapSuccessMsg struct {
	Hosts       []shared.Host
	NeedsUnlock bool
}

type bwBootstrapErrorMsg struct {
	Err error
}

type serverRestartSuccessMsg struct {
	Hosts []shared.Host
}

type serverRestartErrorMsg struct {
	Err error
}

type serverKillSuccessMsg struct{}

type serverKillErrorMsg struct {
	Err error
}

type hostsPollTickMsg struct{}
type unlockPollTickMsg struct{}

func initialModel() model {
	return model{
		display:           components.NewDisplay(),
		clearOnNextRender: true,
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
		m.clearOnNextRender = false
		if msg.NeedsUnlock {
			m.display.SetCurrentView(view.NewBwUnlockView())
			return m, tea.Batch(viewCmd, unlockPollTickCmd())
		}
		m.display.SetCurrentView(view.NewHostsView(msg.Hosts))
		return m, tea.Batch(viewCmd, hostsPollTickCmd())

	case bwBootstrapErrorMsg:
		m.clearOnNextRender = false
		m.display.SetCurrentView(view.NewTextView(msg.Err.Error()))
		return m, viewCmd

	case bwUnlockSuccessMsg:
		m.clearOnNextRender = false
		m.display.SetCurrentView(view.NewHostsView(msg.Hosts))
		return m, tea.Batch(viewCmd, hostsPollTickCmd())

	case bwUnlockErrorMsg:
		m.clearOnNextRender = false
		unlockView := view.NewBwUnlockView()
		if msg.Err != shared.ErrServerExists {
			unlockView.SetError(msg.Err.Error())
		}
		m.display.SetCurrentView(unlockView)
		return m, tea.Batch(viewCmd, unlockPollTickCmd())

	case bwReloadSuccessMsg:
		m.clearOnNextRender = false
		selectedID := ""
		if hostsView, ok := m.display.CurrentView.(*view.HostsView); ok {
			selectedID = hostsView.SelectedHostID()
		}
		m.display.SetCurrentView(view.NewHostsViewWithSelection(msg.Hosts, selectedID))
		return m, viewCmd

	case bwReloadErrorMsg:
		m.clearOnNextRender = true
		spinnerView := view.NewSpinnerView("starting Shellwarden...")
		m.display.SetCurrentView(spinnerView)
		return m, tea.Batch(spinnerView.Init(), bootstrapCmd())

	case bwGetHostsSuccessMsg:
		m.clearOnNextRender = false
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
			m.clearOnNextRender = true
			spinnerView := view.NewSpinnerView("starting Shellwarden...")
			m.display.SetCurrentView(spinnerView)
			return m, tea.Batch(spinnerView.Init(), bootstrapCmd())
		}
		return m, viewCmd

	case serverRestartSuccessMsg:
		m.clearOnNextRender = false
		m.display.SetCurrentView(view.NewHostsView(msg.Hosts))
		return m, tea.Batch(viewCmd, hostsPollTickCmd())

	case serverRestartErrorMsg:
		m.clearOnNextRender = true
		spinnerView := view.NewSpinnerView("starting Shellwarden...")
		m.display.SetCurrentView(spinnerView)
		return m, tea.Batch(spinnerView.Init(), bootstrapCmd())

	case serverKillSuccessMsg:
		m.clearOnNextRender = true
		spinnerView := view.NewSpinnerView("starting Shellwarden...")
		m.display.SetCurrentView(spinnerView)
		return m, tea.Batch(spinnerView.Init(), bootstrapCmd())

	case serverKillErrorMsg:
		m.clearOnNextRender = true
		spinnerView := view.NewSpinnerView("starting Shellwarden...")
		m.display.SetCurrentView(spinnerView)
		return m, tea.Batch(spinnerView.Init(), bootstrapCmd())

	case sshFinishedMsg:
		m.clearOnNextRender = true
		spinnerView := view.NewSpinnerView("starting Shellwarden...")
		m.display.SetCurrentView(spinnerView)
		return m, tea.Batch(spinnerView.Init(), bootstrapCmd())

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

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "ctrl+k":
			if m.display != nil && m.display.CurrentView != nil {
				current := m.display.CurrentView
				m.display.SetCurrentView(view.NewDialogServerRestartView(
					current,
					func() {
						spinnerView := view.NewSpinnerView("restarting Shellwarden...")
						m.display.SetCurrentView(spinnerView)
					},
					func() {
						spinnerView := view.NewSpinnerView("stopping Shellwarden...")
						m.display.SetCurrentView(spinnerView)
					},
				))
			}
			return m, nil
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

			if hostsView, ok := m.display.CurrentView.(*view.HostsView); ok {
				host, found := hostsView.SelectedHost()
				if found {
					return m, connectSSHCmd(host)
				}
				return m, viewCmd
			}

			m.display.CurrentView.OnEnter()
		case "r":
			if _, ok := m.display.CurrentView.(*view.DialogServerRestartView); ok {
				spinnerView := view.NewSpinnerView("restarting Shellwarden...")
				m.display.SetCurrentView(spinnerView)
				return m, tea.Batch(spinnerView.Init(), restartServerCmd())
			}
			if _, ok := m.display.CurrentView.(*view.HostsView); ok {
				spinnerView := view.NewSpinnerView("updating hosts...")
				m.display.SetCurrentView(spinnerView)
				return m, tea.Batch(spinnerView.Init(), reloadHostsCmd())
			}

			m.display.CurrentView.OnKey(msg.String())
		case "k":
			if _, ok := m.display.CurrentView.(*view.DialogServerRestartView); ok {
				spinnerView := view.NewSpinnerView("stopping Shellwarden...")
				m.display.SetCurrentView(spinnerView)
				return m, tea.Batch(spinnerView.Init(), killServerCmd())
			}

			m.display.CurrentView.OnKey(msg.String())
		case "esc":
			m.display.CurrentView.OnEsc()
		default:
			m.display.CurrentView.OnKey(msg.String())
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

	content := m.display.Render()
	if m.clearOnNextRender {
		content = clearTerminalSequence + content
	}

	return tea.NewView(content)
}

func bootstrapCmd() tea.Cmd {
	return func() tea.Msg {
		result, err := Bootstrap()
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
		hosts, err := StartServerWithPassword(password)
		if err != nil {
			return bwUnlockErrorMsg{Err: err}
		}

		return bwUnlockSuccessMsg{Hosts: hosts}
	}
}

func reloadHostsCmd() tea.Cmd {
	return func() tea.Msg {
		hosts, err := ReloadHosts()
		if err != nil {
			return bwReloadErrorMsg{Err: err}
		}

		return bwReloadSuccessMsg{Hosts: hosts}
	}
}

func getHostsCmd() tea.Cmd {
	return func() tea.Msg {
		hosts, err := GetHosts()
		if err != nil {
			return bwGetHostsErrorMsg{Err: err}
		}

		return bwGetHostsSuccessMsg{Hosts: hosts}
	}
}

func restartServerCmd() tea.Cmd {
	return func() tea.Msg {
		session, err := KillServer()
		if err != nil {
			return serverRestartErrorMsg{Err: err}
		}

		hosts, err := RestartServerWithSession(session)
		if err != nil {
			return serverRestartErrorMsg{Err: err}
		}

		return serverRestartSuccessMsg{Hosts: hosts}
	}
}

func killServerCmd() tea.Cmd {
	return func() tea.Msg {
		if _, err := KillServer(); err != nil {
			return serverKillErrorMsg{Err: err}
		}

		waitForServerStop()
		return serverKillSuccessMsg{}
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
