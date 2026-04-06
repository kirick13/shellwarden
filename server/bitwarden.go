package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	"github.com/kirick13/shellwarden/shared"
)

type unlockResult struct {
	Session string
	Hosts   []shared.Host
}

func listHosts(session string) ([]shared.Host, error) {
	itemsJSON, err := listItems(session)
	if err != nil {
		return nil, err
	}

	return parseHosts(itemsJSON)
}

func unlock(password string) (unlockResult, error) {
	session, err := runUnlock(password)
	if err != nil {
		return unlockResult{}, err
	}

	itemsJSON, err := listItems(session)
	if err != nil {
		return unlockResult{}, err
	}

	hosts, err := parseHosts(itemsJSON)
	if err != nil {
		return unlockResult{}, err
	}

	return unlockResult{
		Session: session,
		Hosts:   hosts,
	}, nil
}

func runUnlock(password string) (string, error) {
	cmd := exec.Command("bw", "unlock", "--raw", "--passwordenv", "BW_PASSWORD")
	cmd.Env = append(os.Environ(), "BW_PASSWORD="+password)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", wrapError("unlock failed", msg)
	}

	session := strings.TrimSpace(stdout.String())
	if session == "" {
		return "", wrapError("unlock failed", "empty session returned by bw")
	}

	return session, nil
}

func listItems(session string) (string, error) {
	if err := syncVault(session); err != nil {
		return "", err
	}

	cmd := exec.Command("bw", "list", "items")
	cmd.Env = append(os.Environ(), "BW_SESSION="+session)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", wrapError("listing items failed", msg)
	}

	return stdout.String(), nil
}

func syncVault(session string) error {
	cmd := exec.Command("bw", "sync")
	cmd.Env = append(os.Environ(), "BW_SESSION="+session)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return wrapError("sync failed", msg)
	}

	return nil
}

type bwItem struct {
	ID     string        `json:"id"`
	Type   int           `json:"type"`
	Name   string        `json:"name"`
	Fields []bwItemField `json:"fields"`
}

type bwItemField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func parseHosts(itemsJSON string) ([]shared.Host, error) {
	var items []bwItem
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		return nil, wrapError("parsing items failed", err.Error())
	}

	hosts := make([]shared.Host, 0, len(items))
	for _, item := range items {
		if item.Type != 5 {
			continue
		}

		host := shared.Host{
			ID:   item.ID,
			Name: item.Name,
		}

		for _, field := range item.Fields {
			switch field.Name {
			case "IPv4":
				host.IPv4 = field.Value
			case "SSH port":
				host.SSHPort = field.Value
			case "username":
				host.Username = field.Value
			}
		}

		if host.IPv4 == "" {
			continue
		}

		hosts = append(hosts, host)
	}

	return hosts, nil
}

func wrapError(prefix, msg string) error {
	return shared.Error(prefix + ": " + msg)
}
