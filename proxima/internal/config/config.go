package config

import "time"

// AuthMethod identifies how Proxmox API credentials are supplied.
type AuthMethod string

const (
	AuthPassword AuthMethod = "password"
	AuthToken    AuthMethod = "token"
)

// Auth holds exactly one authentication mode after validation.
type Auth struct {
	Method      AuthMethod
	User        string // password mode
	Password    string // password mode
	TokenID     string // token mode
	TokenSecret string // token mode
}

// Config is the resolved configuration after load, merge, and defaults.
type Config struct {
	ProxmoxHost           string
	ProxmoxPort           int
	SSHPort               int
	Node                  string
	Auth                  Auth
	VMID                  int
	VMName                string
	MemoryMiB             int
	Cores                 int
	DiskSize              string
	RootPassword          string
	GuestSSHPublicKeyPath string
	Storage               string
	Bridge                string
	OutputQcow2           string
	WorkDir               string
	InsecureTLS           bool
	SSHUser               string
	SSHPrivateKeyPath     string
	StartAfterCreate      bool
	WaitIP                bool
	AgentWaitTimeout      time.Duration
	AgentPollInterval     time.Duration
	startAfterCreateSet   bool
	waitIPSet             bool
}

// ApplyDefaults fills zero-valued optional fields with constitution defaults.
func ApplyDefaults(c *Config) {
	if c.ProxmoxPort == 0 {
		c.ProxmoxPort = 8006
	}
	if c.SSHPort == 0 {
		c.SSHPort = 22
	}
	if c.Storage == "" {
		c.Storage = "local-lvm"
	}
	if c.Bridge == "" {
		c.Bridge = "vmbr0"
	}
	if c.OutputQcow2 == "" {
		c.OutputQcow2 = "alpine-ready.qcow2"
	}
	if c.WorkDir == "" {
		c.WorkDir = ".proxima-work"
	}
	if !c.startAfterCreateSet {
		c.StartAfterCreate = true
	}
	if !c.waitIPSet {
		c.WaitIP = true
	}
	if c.AgentWaitTimeout == 0 {
		c.AgentWaitTimeout = 120 * time.Second
	}
	if c.AgentPollInterval == 0 {
		c.AgentPollInterval = 2 * time.Second
	}
}
