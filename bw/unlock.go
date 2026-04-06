package bw

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
)

type UnlockResult struct {
	Session   string
	Bookmarks []Bookmark
}

func ListBookmarks(session string) ([]Bookmark, error) {
	itemsJSON, err := listItems(session)
	if err != nil {
		return nil, err
	}

	return parseBookmarks(itemsJSON)
}

type Bookmark struct {
	ID       string
	Name     string
	IPv4     string
	SSHPort  string
	Username string
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

	bookmarks, err := parseBookmarks(itemsJSON)
	if err != nil {
		return UnlockResult{}, err
	}

	return UnlockResult{
		Session:   session,
		Bookmarks: bookmarks,
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

func syncVault(session string) error {
	cmd := exec.Command("bw", "sync", "--session", session)

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

func parseBookmarks(itemsJSON string) ([]Bookmark, error) {
	var items []bwItem
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		return nil, wrapError("parsing items failed", err.Error())
	}

	bookmarks := make([]Bookmark, 0, len(items))
	for _, item := range items {
		if item.Type != 5 {
			continue
		}

		bookmark := Bookmark{
			ID:   item.ID,
			Name: item.Name,
		}

		for _, field := range item.Fields {
			switch field.Name {
			case "IPv4":
				bookmark.IPv4 = field.Value
			case "SSH port":
				bookmark.SSHPort = field.Value
			case "username":
				bookmark.Username = field.Value
			}
		}

		if bookmark.IPv4 == "" {
			continue
		}

		bookmarks = append(bookmarks, bookmark)
	}

	return bookmarks, nil
}

func wrapError(prefix, msg string) error {
	return Error(prefix + ": " + msg)
}

type Error string

func (e Error) Error() string {
	return string(e)
}
