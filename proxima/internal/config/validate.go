package config

import (
	"errors"
	"regexp"
)

var (
	namePattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	diskSizePattern = regexp.MustCompile(`^[1-9][0-9]*[KMGTP]$`)
)

// Validate checks required fields and invariants; sets Auth.Method when auth is valid.
func Validate(c Config) error {
	if c.ProxmoxHost == "" {
		return errors.New("proxmox_host is required")
	}
	if c.Node == "" {
		return errors.New("node is required")
	}

	hasPassword := c.Auth.User != "" || c.Auth.Password != ""
	hasToken := c.Auth.TokenID != "" || c.Auth.TokenSecret != ""
	switch {
	case !hasPassword && !hasToken:
		return errors.New("authentication is required: provide user/password or token_id/token_secret")
	case hasPassword && hasToken:
		return errors.New("authentication conflict: provide either password auth or token auth, not both")
	case hasPassword:
		if c.Auth.User == "" {
			return errors.New("user is required for password authentication")
		}
		if c.Auth.Password == "" {
			return errors.New("password is required for password authentication")
		}
	case hasToken:
		if c.Auth.TokenID == "" {
			return errors.New("token_id is required for token authentication")
		}
		if c.Auth.TokenSecret == "" {
			return errors.New("token_secret is required for token authentication")
		}
	}

	if c.VMID < 100 || c.VMID > 999999999 {
		return errors.New("vmid must be between 100 and 999999999")
	}
	if c.VMName == "" {
		return errors.New("name is required")
	}
	if !namePattern.MatchString(c.VMName) {
		return errors.New(`name must match ^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	}
	if c.MemoryMiB < 128 {
		return errors.New("memory must be >= 128")
	}
	if c.Cores < 1 {
		return errors.New("cores must be >= 1")
	}
	if c.DiskSize == "" {
		return errors.New("disk_size is required")
	}
	if !diskSizePattern.MatchString(c.DiskSize) {
		return errors.New(`disk_size must match ^[1-9][0-9]*[KMGTP]$`)
	}
	if c.RootPassword == "" {
		return errors.New("root_password is required")
	}
	if c.AgentWaitTimeout <= 0 {
		return errors.New("agent_wait_timeout must be > 0")
	}
	return nil
}
