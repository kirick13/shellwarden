package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"github.com/creack/pty"
	"github.com/muesli/cancelreader"
	"github.com/kirick13/shellwarden/shared"
	keys "github.com/kirick13/shellwarden/ui/components/keys"
)

const clearTerminalSequence = "\x1b[2J\x1b[3J\x1b[H"

type sshFinishedMsg struct {
	Err error
}

type sshExecCommand struct {
	host   shared.Host
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func connectSSHCmd(host shared.Host) tea.Cmd {
	return tea.Exec(&sshExecCommand{host: host}, func(err error) tea.Msg {
		return sshFinishedMsg{Err: err}
	})
}

func (c *sshExecCommand) SetStdin(r io.Reader) {
	c.stdin = r
}

func (c *sshExecCommand) SetStdout(w io.Writer) {
	c.stdout = w
}

func (c *sshExecCommand) SetStderr(w io.Writer) {
	c.stderr = w
}

func (c *sshExecCommand) Run() error {
	stdinFile, _ := c.stdin.(*os.File)
	if stdinFile == nil {
		stdinFile = os.Stdin
	}

	stdoutWriter := c.stdout
	if stdoutWriter == nil {
		stdoutWriter = os.Stdout
	}

	socketPath, err := findBitwardenAgentSocket()
	if err != nil {
		_, _ = fmt.Fprintf(stdoutWriter, "\n%v\n", err)
		return err
	}

	var oldState *term.State
	if stdinFile != nil {
		state, err := term.MakeRaw(stdinFile.Fd())
		if err == nil {
			oldState = state
			defer func() {
				_ = term.Restore(stdinFile.Fd(), oldState)
			}()
		}
	}

	for {
		runErr := c.runSession(stdinFile, stdoutWriter, socketPath)
		if action := c.promptNextAction(stdinFile, stdoutWriter, runErr); action != "reconnect" {
			return nil
		}
	}
}

func (c *sshExecCommand) runSession(stdinFile *os.File, stdoutWriter io.Writer, socketPath string) error {
	cmd := exec.Command("ssh", c.args()...)
	cmd.Env = withEnv(os.Environ(), "SSH_AUTH_SOCK", socketPath)

	_, _ = fmt.Fprintf(stdoutWriter, "\x1b]0;%s\x07", c.host.Name)
	_, _ = fmt.Fprint(stdoutWriter, clearTerminalSequence)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(stdoutWriter, "\nFailed to start ssh: %v\n", err)
		return err
	}
	defer func() { _ = ptmx.Close() }()

	if stdinFile != nil {
		if err := pty.InheritSize(stdinFile, ptmx); err == nil {
			sigch := make(chan os.Signal, 1)
			signal.Notify(sigch, syscall.SIGWINCH)
			defer signal.Stop(sigch)
			go func() {
				for range sigch {
					_ = pty.InheritSize(stdinFile, ptmx)
				}
			}()
			sigch <- syscall.SIGWINCH
		}
	}

	copyDone := make(chan struct{}, 1)
	go func() {
		_, _ = io.Copy(stdoutWriter, ptmx)
		copyDone <- struct{}{}
	}()

	stdinDone := make(chan struct{}, 1)
	cancelStdin := func() {}
	if stdinFile != nil {
		cancelReader, cancelErr := cancelreader.NewReader(stdinFile)
		if cancelErr == nil {
			cancelStdin = func() {
				cancelReader.Cancel()
				_ = cancelReader.Close()
				<-stdinDone
			}
			go func() {
				_, _ = io.Copy(ptmx, cancelReader)
				stdinDone <- struct{}{}
			}()
		} else {
			cancelStdin = func() {}
			go func() {
				_, _ = io.Copy(ptmx, stdinFile)
				stdinDone <- struct{}{}
			}()
		}
	}

	if stdinFile == nil {
		go func() {
			stdinDone <- struct{}{}
		}()
	}

	waitErr := cmd.Wait()
	<-copyDone
	if stdinFile != nil {
		cancelStdin()
	}
	return waitErr
}

func (c *sshExecCommand) promptNextAction(stdinFile *os.File, stdoutWriter io.Writer, runErr error) string {
	_, _ = fmt.Fprint(stdoutWriter, "\n\nSSH session ended.")
	if runErr != nil {
		_, _ = fmt.Fprintf(stdoutWriter, " (%v)", runErr)
	}
	_, _ = fmt.Fprintf(stdoutWriter, "\r\n%s\r\n", keys.RenderKeys([]keys.Keys{
		{Key: "enter", Title: "reconnect"},
		{Key: "q", Title: "quit"},
	}))

	if stdinFile == nil {
		return "quit"
	}

	buf := make([]byte, 1)
	for {
		_, err := stdinFile.Read(buf)
		if err != nil {
			return "quit"
		}

		switch buf[0] {
		case '\r', '\n':
			_, _ = fmt.Fprint(stdoutWriter, "\r\n")
			return "reconnect"
		case 'q', 'Q':
			_, _ = fmt.Fprint(stdoutWriter, "\r\n")
			return "quit"
		}
	}
}

func (c *sshExecCommand) args() []string {
	args := make([]string, 0, 4)
	if c.host.SSHPort != "" {
		args = append(args, "-p", c.host.SSHPort)
	}
	args = append(args, sshTarget(c.host))
	return args
}

func sshTarget(host shared.Host) string {
	if host.Username != "" {
		return host.Username + "@" + host.IPv4
	}
	return host.IPv4
}

func findBitwardenAgentSocket() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	candidates := []string{
		filepath.Join(homeDir, ".bitwarden-ssh-agent.sock"),
		filepath.Join(homeDir, "Library", "Containers", "com.bitwarden.desktop", "Data", ".bitwarden-ssh-agent.sock"),
		filepath.Join(homeDir, ".var", "app", "com.bitwarden.desktop", "data", ".bitwarden-ssh-agent.sock"),
		filepath.Join(homeDir, "snap", "bitwarden", "current", ".bitwarden-ssh-agent.sock"),
	}

	if envSock := os.Getenv("SSH_AUTH_SOCK"); filepath.Base(envSock) == ".bitwarden-ssh-agent.sock" {
		candidates = append([]string{envSock}, candidates...)
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && info.Mode()&os.ModeSocket != 0 {
			return candidate, nil
		}
	}

	return "", shared.Error("Bitwarden SSH agent socket not found. Enable Bitwarden desktop SSH agent first.")
}

func withEnv(env []string, key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	replaced := false
	for _, entry := range env {
		if len(entry) >= len(prefix) && entry[:len(prefix)] == prefix {
			out = append(out, prefix+value)
			replaced = true
			continue
		}
		out = append(out, entry)
	}
	if !replaced {
		out = append(out, prefix+value)
	}
	return out
}
