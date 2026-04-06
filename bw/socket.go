package bw

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type BootstrapResult struct {
	Hosts       []Host
	NeedsUnlock bool
}

type request struct {
	Type     string `json:"type"`
	Password string `json:"password,omitempty"`
}

type response struct {
	Type    string `json:"type"`
	Hosts   []Host `json:"hosts,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

const (
	pingTimeout      = 100 * time.Millisecond
	getHostsTimeout  = 500 * time.Millisecond
	unlockTimeout    = 45 * time.Second
	reloadTimeout    = 30 * time.Second
)

var (
	serverMu      sync.RWMutex
	serverSession string
	serverHosts   []Host
	serverLn      net.Listener
)

var ErrServerExists = Error("server already exists")

func Bootstrap() (BootstrapResult, error) {
	if _, err := Ping(pingTimeout); err == nil {
		hosts, err := GetHosts()
		if err == nil {
			return BootstrapResult{Hosts: hosts}, nil
		}
		if errors.Is(err, ErrLocked) {
			return BootstrapResult{NeedsUnlock: true}, nil
		}
	}

	return BootstrapResult{NeedsUnlock: true}, nil
}

func Ping(timeout time.Duration) (bool, error) {
	resp, err := doRequest(request{Type: "ping"}, timeout)
	if err != nil {
		return false, err
	}

	if resp.Type != "pong" {
		return false, fmt.Errorf("unexpected ping response: %s", resp.Type)
	}

	return true, nil
}

func GetHosts() ([]Host, error) {
	resp, err := doRequest(request{Type: "getHosts"}, getHostsTimeout)
	if err != nil {
		return nil, err
	}

	return decodeHostsResponse(resp)
}

func ReloadHosts() ([]Host, error) {
	resp, err := doRequest(request{Type: "reloadHosts"}, reloadTimeout)
	if err != nil {
		return nil, err
	}

	return decodeHostsResponse(resp)
}

func UnlockHosts(password string) ([]Host, error) {
	started, err := ensureServer()
	if err != nil {
		return nil, err
	}
	if !started {
		return nil, ErrServerExists
	}

	resp, err := doRequest(request{Type: "unlock", Password: password}, unlockTimeout)
	if err != nil {
		stopServer()
		return nil, err
	}

	hosts, err := decodeHostsResponse(resp)
	if err != nil {
		stopServer()
		return nil, err
	}

	return hosts, nil
}

func decodeHostsResponse(resp response) ([]Host, error) {
	switch resp.Type {
	case "hosts":
		return resp.Hosts, nil
	case "error":
		if resp.Code == "locked" {
			return nil, ErrLocked
		}
		if resp.Code == "auth" {
			return nil, ErrAuth
		}
		if resp.Message != "" {
			return nil, Error(resp.Message)
		}
		return nil, Error("unknown socket error")
	default:
		return nil, Error("unexpected socket response")
	}
}

func ensureServer() (bool, error) {
	serverMu.Lock()
	if serverLn != nil {
		serverMu.Unlock()
		return false, nil
	}
	serverMu.Unlock()

	path, err := socketPath()
	if err != nil {
		return false, err
	}

	if st, err := os.Stat(path); err == nil && (st.Mode()&os.ModeSocket) != 0 {
		conn, dialErr := net.DialTimeout("unix", path, pingTimeout)
		if dialErr != nil {
			_ = os.Remove(path)
		} else {
			_ = conn.Close()
		}
	}

	ln, err := net.Listen("unix", path)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) || strings.Contains(err.Error(), "address already in use") {
			return false, nil
		}
		return false, err
	}

	if err := os.Chmod(path, 0o600); err != nil {
		_ = ln.Close()
		_ = os.Remove(path)
		return false, err
	}

	serverMu.Lock()
	serverLn = ln
	serverMu.Unlock()

	go serve(ln, path)

	return true, nil
}

func serve(ln net.Listener, path string) {
	defer func() {
		_ = ln.Close()
		_ = os.Remove(path)

		serverMu.Lock()
		if serverLn == ln {
			serverLn = nil
			serverSession = ""
			serverHosts = nil
		}
		serverMu.Unlock()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	var req request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		_ = json.NewEncoder(conn).Encode(response{Type: "error", Code: "bad_request", Message: err.Error()})
		return
	}

	resp := handleRequest(req)
	_ = json.NewEncoder(conn).Encode(resp)

	if resp.Code == "auth" {
		go stopServer()
	}
}

func handleRequest(req request) response {
	switch req.Type {
	case "ping":
		return response{Type: "pong"}
	case "getHosts":
		serverMu.RLock()
		defer serverMu.RUnlock()
		if serverSession == "" {
			return response{Type: "error", Code: "locked", Message: "Bitwarden is locked"}
		}
		return response{Type: "hosts", Hosts: append([]Host(nil), serverHosts...)}
	case "reloadHosts":
		serverMu.RLock()
		session := serverSession
		serverMu.RUnlock()
		if session == "" {
			return response{Type: "error", Code: "locked", Message: "Bitwarden is locked"}
		}

		hosts, err := ListHosts(session)
		if err != nil {
			if isAuthError(err) {
				return response{Type: "error", Code: "auth", Message: err.Error()}
			}
			return response{Type: "error", Code: "reload_failed", Message: err.Error()}
		}

		serverMu.Lock()
		serverHosts = append([]Host(nil), hosts...)
		serverMu.Unlock()
		return response{Type: "hosts", Hosts: hosts}
	case "unlock":
		result, err := Unlock(req.Password)
		if err != nil {
			if isAuthError(err) {
				return response{Type: "error", Code: "auth", Message: err.Error()}
			}
			return response{Type: "error", Code: "unlock_failed", Message: err.Error()}
		}

		serverMu.Lock()
		serverSession = result.Session
		serverHosts = append([]Host(nil), result.Hosts...)
		serverMu.Unlock()
		return response{Type: "hosts", Hosts: result.Hosts}
	default:
		return response{Type: "error", Code: "unknown_request", Message: "unknown request type"}
	}
}

func stopServer() {
	serverMu.Lock()
	ln := serverLn
	serverMu.Unlock()
	if ln != nil {
		_ = ln.Close()
	}
}

func doRequest(req request, timeout time.Duration) (response, error) {
	path, err := socketPath()
	if err != nil {
		return response{}, err
	}

	conn, err := net.DialTimeout("unix", path, timeout)
	if err != nil {
		return response{}, err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return response{}, err
	}

	var resp response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return response{}, err
	}

	return resp, nil
}

func socketPath() (string, error) {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "shellwarden.sock"), nil
	}

	return filepath.Join(os.TempDir(), fmt.Sprintf("shellwarden-%d.sock", os.Getuid())), nil
}

var (
	ErrLocked = Error("bitwarden locked")
	ErrAuth   = Error("bitwarden auth error")
)

func isAuthError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "locked") ||
		strings.Contains(msg, "not logged in") ||
		strings.Contains(msg, "invalid session") ||
		strings.Contains(msg, "unauthorized")
}
