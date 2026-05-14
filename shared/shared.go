package shared

import (
	"fmt"
	"os"
	"path/filepath"
)

type Host struct {
	ID       string
	Name     string
	IP       string
	SSHPort  string
	SSHPublicKey string
	Username string
}

type Error string

func (e Error) Error() string {
	return string(e)
}

type Request struct {
	Type   string `json:"type"`
	HostID string `json:"hostId,omitempty"`
}

type Response struct {
	Type          string `json:"type"`
	Hosts         []Host `json:"hosts,omitempty"`
	Session       string `json:"session,omitempty"`
	SSHPrivateKey string `json:"sshPrivateKey,omitempty"`
	Code          string `json:"code,omitempty"`
	Message       string `json:"message,omitempty"`
}

var (
	ErrLocked       = Error("bitwarden locked")
	ErrAuth         = Error("bitwarden auth error")
	ErrServerExists = Error("server already exists")
)

func SocketPath() (string, error) {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "shellwarden.sock"), nil
	}

	return filepath.Join(os.TempDir(), fmt.Sprintf("shellwarden-%d.sock", os.Getuid())), nil
}
