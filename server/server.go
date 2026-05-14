package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kirick13/shellwarden/shared"
)

type startupPayload struct {
	Password string `json:"password,omitempty"`
	Session  string `json:"session,omitempty"`
}

type daemon struct {
	mu          sync.RWMutex
	session     string
	hosts       []shared.Host
	privateKeys map[string]string
	ln          net.Listener
	path        string
}

func Run() error {
	payload, err := readStartupPayload(os.Stdin)
	if err != nil {
		return err
	}

	path, err := shared.SocketPath()
	if err != nil {
		return err
	}

	if err := cleanupStaleSocket(path); err != nil {
		return err
	}

	ln, err := net.Listen("unix", path)
	if err != nil {
		return err
	}

	if err := os.Chmod(path, 0o600); err != nil {
		_ = ln.Close()
		_ = os.Remove(path)
		return err
	}

	d := &daemon{
		ln:   ln,
		path: path,
	}

	if err := d.initialize(payload); err != nil {
		d.close()
		return err
	}

	return d.serve()
}

func StartWithPassword(password string) ([]shared.Host, error) {
	if strings.TrimSpace(password) == "" {
		return nil, shared.Error("password is required")
	}

	return startAndWait(startupPayload{Password: password})
}

func RestartWithSession(session string) ([]shared.Host, error) {
	if strings.TrimSpace(session) == "" {
		return nil, shared.Error("session is required")
	}

	waitForServerStop()
	return startAndWait(startupPayload{Session: session})
}

func startAndWait(payload startupPayload) ([]shared.Host, error) {
	if err := spawnDetached(payload); err != nil {
		if !errors.Is(err, shared.ErrServerExists) {
			return nil, err
		}
	}

	return waitForHosts(45 * time.Second)
}

func spawnDetached(payload startupPayload) error {
	cmd, stdin, err := buildServerCommand()
	if err != nil {
		return err
	}

	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return err
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	if err := json.NewEncoder(stdin).Encode(payload); err != nil {
		_ = stdin.Close()
		return err
	}
	_ = stdin.Close()

	select {
	case err := <-waitCh:
		if err == nil {
			return shared.Error("server exited before becoming ready")
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		if strings.Contains(msg, "address already in use") {
			return shared.ErrServerExists
		}
		return shared.Error(msg)
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

func buildServerCommand() (*exec.Cmd, io.WriteCloser, error) {
	if bin := os.Getenv("SHELLWARDEN_SERVER_BIN"); bin != "" {
		cmd := exec.Command(bin)
		stdin, err := cmd.StdinPipe()
		return cmd, stdin, err
	}

	exe, err := os.Executable()
	if err == nil {
		sibling := filepath.Join(filepath.Dir(exe), "shellwarden-server")
		if _, statErr := os.Stat(sibling); statErr == nil {
			cmd := exec.Command(sibling)
			stdin, pipeErr := cmd.StdinPipe()
			return cmd, stdin, pipeErr
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		if _, statErr := os.Stat(filepath.Join(cwd, "go.mod")); statErr == nil {
			cmd := exec.Command("go", "run", "./server")
			cmd.Dir = cwd
			stdin, pipeErr := cmd.StdinPipe()
			return cmd, stdin, pipeErr
		}
	}

	cmd := exec.Command("shellwarden-server")
	stdin, err := cmd.StdinPipe()
	return cmd, stdin, err
}

func waitForHosts(timeout time.Duration) ([]shared.Host, error) {
	deadline := time.Now().Add(timeout)
	for {
		hosts, err := getHostsFromSocket()
		if err == nil {
			return hosts, nil
		}

		if time.Now().After(deadline) {
			return nil, err
		}

		time.Sleep(150 * time.Millisecond)
	}
}

func waitForServerStop() {
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := pingSocket(100 * time.Millisecond); err != nil {
			return
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func readStartupPayload(r io.Reader) (startupPayload, error) {
	var payload startupPayload
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return startupPayload{}, err
	}

	if strings.TrimSpace(payload.Password) == "" && strings.TrimSpace(payload.Session) == "" {
		return startupPayload{}, shared.Error("server startup requires password or session")
	}

	return payload, nil
}

func cleanupStaleSocket(path string) error {
	if st, err := os.Stat(path); err == nil && (st.Mode()&os.ModeSocket) != 0 {
		conn, dialErr := net.DialTimeout("unix", path, 100*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			return shared.ErrServerExists
		}
		return os.Remove(path)
	}

	return nil
}

func (d *daemon) initialize(payload startupPayload) error {
	if payload.Session != "" {
		data, err := listHostData(payload.Session)
		if err != nil {
			return err
		}
		d.mu.Lock()
		d.session = payload.Session
		d.hosts = append([]shared.Host(nil), data.Hosts...)
		d.privateKeys = clonePrivateKeys(data.PrivateKeys)
		d.mu.Unlock()
		return nil
	}

	result, err := unlock(payload.Password)
	if err != nil {
		return err
	}

	d.mu.Lock()
	d.session = result.Session
	d.hosts = append([]shared.Host(nil), result.Hosts...)
	d.privateKeys = clonePrivateKeys(result.PrivateKeys)
	d.mu.Unlock()
	return nil
}

func (d *daemon) serve() error {
	defer d.close()

	for {
		conn, err := d.ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}

		go d.handleConn(conn)
	}
}

func (d *daemon) close() {
	if d.ln != nil {
		_ = d.ln.Close()
	}
	if d.path != "" {
		_ = os.Remove(d.path)
	}
}

func (d *daemon) handleConn(conn net.Conn) {
	defer conn.Close()

	var req shared.Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		_ = json.NewEncoder(conn).Encode(shared.Response{Type: "error", Code: "bad_request", Message: err.Error()})
		return
	}

	resp, shouldStop := d.handleRequest(req)
	_ = json.NewEncoder(conn).Encode(resp)

	if shouldStop {
		go d.stop()
	}
}

func (d *daemon) handleRequest(req shared.Request) (shared.Response, bool) {
	switch req.Type {
	case "ping":
		return shared.Response{Type: "pong"}, false
	case "getHosts":
		d.mu.RLock()
		defer d.mu.RUnlock()
		return shared.Response{Type: "hosts", Hosts: append([]shared.Host(nil), d.hosts...)}, false
	case "reloadHosts":
		d.mu.RLock()
		session := d.session
		d.mu.RUnlock()

		data, err := listHostData(session)
		if err != nil {
			if isAuthError(err) {
				return shared.Response{Type: "error", Code: "auth", Message: err.Error()}, true
			}
			return shared.Response{Type: "error", Code: "reload_failed", Message: err.Error()}, false
		}

		d.mu.Lock()
		d.hosts = append([]shared.Host(nil), data.Hosts...)
		d.privateKeys = clonePrivateKeys(data.PrivateKeys)
		d.mu.Unlock()
		return shared.Response{Type: "hosts", Hosts: data.Hosts}, false
	case "getHostKey":
		d.mu.RLock()
		privateKey, ok := d.privateKeys[req.HostID]
		d.mu.RUnlock()
		if !ok || strings.TrimSpace(privateKey) == "" {
			return shared.Response{Type: "error", Code: "missing_host_key", Message: "SSH private key not found for host"}, false
		}
		return shared.Response{Type: "hostKey", SSHPrivateKey: privateKey}, false
	case "kill":
		d.mu.RLock()
		session := d.session
		d.mu.RUnlock()
		if strings.TrimSpace(session) == "" {
			return shared.Response{Type: "error", Code: "locked", Message: "Bitwarden is locked"}, false
		}
		return shared.Response{Type: "session", Session: session}, true
	default:
		return shared.Response{Type: "error", Code: "unknown_request", Message: "unknown request type"}, false
	}
}

func (d *daemon) stop() {
	if d.ln != nil {
		_ = d.ln.Close()
	}
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()
	return strings.Contains(msg, "invalid master password") ||
		strings.Contains(msg, "vault is locked") ||
		strings.Contains(msg, "You are not logged in") ||
		strings.Contains(msg, "session is not valid") ||
		errors.Is(err, shared.ErrAuth)
}

func clonePrivateKeys(keys map[string]string) map[string]string {
	if len(keys) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(keys))
	for hostID, privateKey := range keys {
		cloned[hostID] = privateKey
	}
	return cloned
}
