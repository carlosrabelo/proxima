package config_test

import (
	"testing"

	"proxima/proxima/internal/config"
)

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
