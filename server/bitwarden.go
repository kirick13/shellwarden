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
	Session     string
	Hosts       []shared.Host
	PrivateKeys map[string]string
}

type hostData struct {
	Hosts       []shared.Host
	PrivateKeys map[string]string
}

func listHosts(session string) ([]shared.Host, error) {
	data, err := listHostData(session)
	if err != nil {
		return nil, err
	}

	return data.Hosts, nil
}

func listHostData(session string) (hostData, error) {
	itemsJSON, err := listItems(session)
	if err != nil {
		return hostData{}, err
	}

	return parseHostData(itemsJSON)
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

	data, err := parseHostData(itemsJSON)
	if err != nil {
		return unlockResult{}, err
	}

	return unlockResult{
		Session:     session,
		Hosts:       data.Hosts,
		PrivateKeys: data.PrivateKeys,
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
	SSHKey *bwItemSSHKey `json:"sshKey"`
}

type bwItemField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type bwItemSSHKey struct {
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

func parseHostData(itemsJSON string) (hostData, error) {
	var items []bwItem
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		return hostData{}, wrapError("parsing items failed", err.Error())
	}

	hosts := make([]shared.Host, 0, len(items))
	privateKeys := make(map[string]string)
	for _, item := range items {
		if item.Type != 5 {
			continue
		}

		host := shared.Host{
			ID:   item.ID,
			Name: item.Name,
		}
		if item.SSHKey != nil {
			host.SSHPublicKey = strings.TrimSpace(item.SSHKey.PublicKey)
		}
		privateKey := ""
		if item.SSHKey != nil {
			privateKey = strings.TrimSpace(item.SSHKey.PrivateKey)
		}

		for _, field := range item.Fields {
			switch field.Name {
			case "IP":
				host.IP = field.Value
			case "SSH port":
				host.SSHPort = field.Value
			case "username":
				host.Username = field.Value
			}
		}

		if host.IP == "" || host.SSHPublicKey == "" || privateKey == "" {
			continue
		}

		hosts = append(hosts, host)
		privateKeys[host.ID] = privateKey
	}

	return hostData{
		Hosts:       hosts,
		PrivateKeys: privateKeys,
	}, nil
}

func wrapError(prefix, msg string) error {
	return shared.Error(prefix + ": " + msg)
}
