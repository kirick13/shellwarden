package bw

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

type UnlockResult struct {
	Session   string
	ItemsJSON string
}

func Unlock(password string) (UnlockResult, error) {
	session, err := runUnlock(password)
	if err != nil {
		return UnlockResult{}, err
	}

	itemsJSON, err := listItems(session)
	if err != nil {
		return UnlockResult{}, err
	}

	return UnlockResult{
		Session:   session,
		ItemsJSON: itemsJSON,
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
	cmd := exec.Command("bw", "list", "items", "--session", session)

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

func wrapError(prefix, msg string) error {
	return Error(prefix + ": " + msg)
}

type Error string

func (e Error) Error() string {
	return string(e)
}
