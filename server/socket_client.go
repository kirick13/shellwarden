package main

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/kirick13/shellwarden/shared"
)

func pingSocket(timeout time.Duration) (bool, error) {
	resp, err := doRequest(shared.Request{Type: "ping"}, timeout)
	if err != nil {
		return false, err
	}

	if resp.Type != "pong" {
		return false, fmt.Errorf("unexpected ping response: %s", resp.Type)
	}

	return true, nil
}

func getHostsFromSocket() ([]shared.Host, error) {
	resp, err := doRequest(shared.Request{Type: "getHosts"}, 500*time.Millisecond)
	if err != nil {
		return nil, err
	}

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
