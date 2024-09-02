package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Override carries optional CLI/env values that win over file contents.
type Override struct{}

type fileConfig struct {
	ProxmoxHost  string `json:"proxmox_host" yaml:"proxmox_host"`
	ProxmoxPort  int    `json:"proxmox_port" yaml:"proxmox_port"`
	SSHPort      int    `json:"ssh_port" yaml:"ssh_port"`
	Node         string `json:"node" yaml:"node"`
	User         string `json:"user" yaml:"user"`
	Password     string `json:"password" yaml:"password"`
	TokenID      string `json:"token_id" yaml:"token_id"`
	TokenSecret  string `json:"token_secret" yaml:"token_secret"`
	VMID         int    `json:"vmid" yaml:"vmid"`
	Name         string `json:"name" yaml:"name"`
	Memory       int    `json:"memory" yaml:"memory"`
	Cores        int    `json:"cores" yaml:"cores"`
	DiskSize     string `json:"disk_size" yaml:"disk_size"`
	RootPassword string `json:"root_password" yaml:"root_password"`
	Storage      string `json:"storage" yaml:"storage"`
	Bridge       string `json:"bridge" yaml:"bridge"`
	Output       string `json:"output" yaml:"output"`
	WorkDir      string `json:"work_dir" yaml:"work_dir"`
	InsecureTLS  bool   `json:"insecure_tls" yaml:"insecure_tls"`
}

// Load reads a YAML or JSON config file, applies overrides and defaults.
func Load(path string, overrides Override) (Config, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml", ".json":
	default:
		return Config{}, fmt.Errorf("config file must end in .yaml, .yml, or .json")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("cannot load config: %s", err)
	}

	var fc fileConfig
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &fc); err != nil {
			return Config{}, fmt.Errorf("cannot load config: %s", err)
		}
	default:
		if err := yaml.Unmarshal(data, &fc); err != nil {
			return Config{}, fmt.Errorf("cannot load config: %s", err)
		}
	}

	c := configFromFile(fc)
	_ = overrides
	ApplyDefaults(&c)
	return c, nil
}

func configFromFile(fc fileConfig) Config {
	return Config{
		ProxmoxHost: fc.ProxmoxHost,
		ProxmoxPort: fc.ProxmoxPort,
		SSHPort:     fc.SSHPort,
		Node:        fc.Node,
		Auth: Auth{
			User:        fc.User,
			Password:    fc.Password,
			TokenID:     fc.TokenID,
			TokenSecret: fc.TokenSecret,
		},
		VMID:         fc.VMID,
		VMName:       fc.Name,
		MemoryMiB:    fc.Memory,
		Cores:        fc.Cores,
		DiskSize:     fc.DiskSize,
		RootPassword: fc.RootPassword,
		Storage:      fc.Storage,
		Bridge:       fc.Bridge,
		OutputQcow2:  fc.Output,
		WorkDir:      fc.WorkDir,
		InsecureTLS:  fc.InsecureTLS,
	}
}
