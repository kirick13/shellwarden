package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/kirick13/shellwarden/shared"
)

type startupPayload struct {
	Password string `json:"password,omitempty"`
	Session  string `json:"session,omitempty"`
}

func StartServerWithPassword(password string) ([]shared.Host, error) {
	if strings.TrimSpace(password) == "" {
		return nil, shared.Error("password is required")
	}

	return startServerAndWait(startupPayload{Password: password})
}

func RestartServerWithSession(session string) ([]shared.Host, error) {
	if strings.TrimSpace(session) == "" {
		return nil, shared.Error("session is required")
	}

	waitForServerStop()
	return startServerAndWait(startupPayload{Session: session})
}

func startServerAndWait(payload startupPayload) ([]shared.Host, error) {
	if err := spawnDetachedServer(payload); err != nil {
		if !errors.Is(err, shared.ErrServerExists) {
			return nil, err
		}
	}

	return waitForHosts(45 * time.Second)
}

func spawnDetachedServer(payload startupPayload) error {
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
		hosts, err := GetHosts()
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
		if _, err := Ping(100 * time.Millisecond); err != nil {
			return
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
