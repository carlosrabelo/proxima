package script

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadPublicKeyFile loads an OpenSSH public key file (typically *.pub).
// A leading "~/" is expanded to the user home directory.
func ReadPublicKeyFile(path string) (string, error) {
	expanded, err := ExpandPath(path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(expanded)
	if err != nil {
		return "", fmt.Errorf("guest ssh public key: %w", err)
	}
	key := normalizeAuthorizedKey(string(data))
	if key == "" {
		return "", fmt.Errorf("guest ssh public key: empty file: %s", expanded)
	}
	if !strings.HasPrefix(key, "ssh-") && !strings.HasPrefix(key, "ecdsa-") && !strings.HasPrefix(key, "sk-") {
		return "", fmt.Errorf("guest ssh public key: unrecognized format in %s", expanded)
	}
	return key, nil
}

// ExpandPath expands a leading "~/".
func ExpandPath(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
