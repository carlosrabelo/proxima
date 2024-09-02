package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"proxima/proxima/internal/config"
)

func TestLoadYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proxima.yaml")
	content := `
proxmox_host: "192.168.1.10"
node: "pve"
user: "root@pam"
password: "secret"
vmid: 200
name: "alpine-docker"
memory: 2048
cores: 2
disk_size: "8G"
root_password: "change-me"
unknown_key: "ignored"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path, config.Override{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ProxmoxHost != "192.168.1.10" {
		t.Fatalf("ProxmoxHost = %q", cfg.ProxmoxHost)
	}
	if cfg.Node != "pve" {
		t.Fatalf("Node = %q", cfg.Node)
	}
	if cfg.Auth.User != "root@pam" || cfg.Auth.Password != "secret" {
		t.Fatalf("Auth = %+v", cfg.Auth)
	}
	if cfg.VMID != 200 || cfg.VMName != "alpine-docker" {
		t.Fatalf("VMID=%d VMName=%q", cfg.VMID, cfg.VMName)
	}
	if cfg.MemoryMiB != 2048 || cfg.Cores != 2 || cfg.DiskSize != "8G" {
		t.Fatalf("Memory=%d Cores=%d DiskSize=%q", cfg.MemoryMiB, cfg.Cores, cfg.DiskSize)
	}
	if cfg.RootPassword != "change-me" {
		t.Fatalf("RootPassword = %q", cfg.RootPassword)
	}
	if cfg.ProxmoxPort != 8006 || cfg.Storage != "local-lvm" || cfg.Bridge != "vmbr0" {
		t.Fatalf("defaults not applied: port=%d storage=%q bridge=%q", cfg.ProxmoxPort, cfg.Storage, cfg.Bridge)
	}
}

func TestLoadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proxima.json")
	content := `{
  "proxmox_host": "10.0.0.5",
  "node": "node1",
  "token_id": "root@pam!proxima",
  "token_secret": "uuid-secret",
  "vmid": 300,
  "name": "json-vm",
  "memory": 1024,
  "cores": 1,
  "disk_size": "4G",
  "root_password": "root-secret",
  "extra": true
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path, config.Override{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ProxmoxHost != "10.0.0.5" || cfg.Node != "node1" {
		t.Fatalf("host=%q node=%q", cfg.ProxmoxHost, cfg.Node)
	}
	if cfg.Auth.TokenID != "root@pam!proxima" || cfg.Auth.TokenSecret != "uuid-secret" {
		t.Fatalf("Auth = %+v", cfg.Auth)
	}
	if cfg.VMID != 300 || cfg.VMName != "json-vm" {
		t.Fatalf("VMID=%d VMName=%q", cfg.VMID, cfg.VMName)
	}
	if cfg.OutputQcow2 != "alpine-ready.qcow2" || cfg.WorkDir != ".proxima-work" {
		t.Fatalf("defaults: output=%q work_dir=%q", cfg.OutputQcow2, cfg.WorkDir)
	}
}

func TestApplyDefaults(t *testing.T) {
	c := config.Config{}
	config.ApplyDefaults(&c)

	if c.ProxmoxPort != 8006 {
		t.Fatalf("ProxmoxPort = %d, want 8006", c.ProxmoxPort)
	}
	if c.SSHPort != 22 {
		t.Fatalf("SSHPort = %d, want 22", c.SSHPort)
	}
	if c.Storage != "local-lvm" {
		t.Fatalf("Storage = %q, want %q", c.Storage, "local-lvm")
	}
	if c.Bridge != "vmbr0" {
		t.Fatalf("Bridge = %q, want %q", c.Bridge, "vmbr0")
	}
	if c.OutputQcow2 != "alpine-ready.qcow2" {
		t.Fatalf("OutputQcow2 = %q, want %q", c.OutputQcow2, "alpine-ready.qcow2")
	}
	if c.WorkDir != ".proxima-work" {
		t.Fatalf("WorkDir = %q, want %q", c.WorkDir, ".proxima-work")
	}
	if c.InsecureTLS {
		t.Fatal("InsecureTLS = true, want false")
	}
}

func TestApplyDefaultsPreservesSetValues(t *testing.T) {
	c := config.Config{
		ProxmoxPort: 443,
		SSHPort:     2222,
		Storage:     "ceph",
		Bridge:      "vmbr1",
		OutputQcow2: "custom.qcow2",
		WorkDir:     "/tmp/work",
		InsecureTLS: true,
	}
	config.ApplyDefaults(&c)

	if c.ProxmoxPort != 443 {
		t.Fatalf("ProxmoxPort = %d, want 443", c.ProxmoxPort)
	}
	if c.SSHPort != 2222 {
		t.Fatalf("SSHPort = %d, want 2222", c.SSHPort)
	}
	if c.Storage != "ceph" {
		t.Fatalf("Storage = %q, want %q", c.Storage, "ceph")
	}
	if c.Bridge != "vmbr1" {
		t.Fatalf("Bridge = %q, want %q", c.Bridge, "vmbr1")
	}
	if c.OutputQcow2 != "custom.qcow2" {
		t.Fatalf("OutputQcow2 = %q, want %q", c.OutputQcow2, "custom.qcow2")
	}
	if c.WorkDir != "/tmp/work" {
		t.Fatalf("WorkDir = %q, want %q", c.WorkDir, "/tmp/work")
	}
	if !c.InsecureTLS {
		t.Fatal("InsecureTLS = false, want true")
	}
}
