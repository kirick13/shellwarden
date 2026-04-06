package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/kirick13/shellwarden/shared"
)

type BootstrapResult struct {
	Hosts       []shared.Host
	NeedsUnlock bool
}

const (
	pingTimeout     = 100 * time.Millisecond
	getHostsTimeout = 500 * time.Millisecond
	reloadTimeout   = 30 * time.Second
	killTimeout     = 2 * time.Second
)

func Bootstrap() (BootstrapResult, error) {
	if _, err := Ping(pingTimeout); err == nil {
		hosts, err := GetHosts()
		if err == nil {
			return BootstrapResult{Hosts: hosts}, nil
		}
		if errors.Is(err, shared.ErrLocked) {
			return BootstrapResult{NeedsUnlock: true}, nil
		}
	}

	return BootstrapResult{NeedsUnlock: true}, nil
}

func Ping(timeout time.Duration) (bool, error) {
	resp, err := doRequest(shared.Request{Type: "ping"}, timeout)
	if err != nil {
		return false, err
	}

	if resp.Type != "pong" {
		return false, fmt.Errorf("unexpected ping response: %s", resp.Type)
	}

	return true, nil
}

func GetHosts() ([]shared.Host, error) {
	resp, err := doRequest(shared.Request{Type: "getHosts"}, getHostsTimeout)
	if err != nil {
		return nil, err
	}

	return decodeHostsResponse(resp)
}

func ReloadHosts() ([]shared.Host, error) {
	resp, err := doRequest(shared.Request{Type: "reloadHosts"}, reloadTimeout)
	if err != nil {
		return nil, err
	}

	return decodeHostsResponse(resp)
}

func KillServer() (string, error) {
	resp, err := doRequest(shared.Request{Type: "kill"}, killTimeout)
	if err != nil {
		return "", err
	}

	switch resp.Type {
	case "session":
		if resp.Session == "" {
			return "", shared.Error("server returned empty session")
		}
		return resp.Session, nil
	case "error":
		if resp.Code == "locked" {
			return "", shared.ErrLocked
		}
		if resp.Code == "auth" {
			return "", shared.ErrAuth
		}
		if resp.Message != "" {
			return "", shared.Error(resp.Message)
		}
		return "", shared.Error("unknown socket error")
	default:
		return "", shared.Error("unexpected socket response")
	}
}

func decodeHostsResponse(resp shared.Response) ([]shared.Host, error) {
	switch resp.Type {
	case "hosts":
		return resp.Hosts, nil
	case "error":
		if resp.Code == "locked" {
			return nil, shared.ErrLocked
		}
		if resp.Code == "auth" {
			return nil, shared.ErrAuth
		}
		if resp.Message != "" {
			return nil, shared.Error(resp.Message)
		}
		return nil, shared.Error("unknown socket error")
	default:
		return nil, shared.Error("unexpected socket response")
	}
}

func doRequest(req shared.Request, timeout time.Duration) (shared.Response, error) {
	path, err := shared.SocketPath()
	if err != nil {
		return shared.Response{}, err
	}

	conn, err := net.DialTimeout("unix", path, timeout)
	if err != nil {
		return shared.Response{}, err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return shared.Response{}, err
	}

	var resp shared.Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return shared.Response{}, err
	}

	return resp, nil
}
