package config

import (
	"os"
	"testing"
)

func TestConfigAdapter_LoadConfig(t *testing.T) {
	// Create temporary config file
	configContent := `
proxmox:
  host: "test-host"
  port: 8006
  user: "test@pam"
  password: "test-password"
  node: "test-node"

ssh:
  user: "root"
  password: "ssh-password"
  key_path: ""
  port: 22
  copy_local_key: false

vms:
  - name: "test-vm"
    vmid: 100
    cores: 2
    memory: 2048
    disk_size: "20G"
    network:
      bridge: "vmbr0"
      vlan: 100
      model: "virtio"
    os_template: "local:vztmpl/ubuntu-22.04-standard_22.04-1_amd64.tar.zst"
    tags: ["test"]
    auto_start: true
    ssh:
      user: "ubuntu"
      password: "ubuntu-password"
      key_path: ""
      copy_local_key: true
      authorized_keys: []
    scripts:
      - name: "test-script"
        path: "/test/script.sh"
        args: ["--test"]
        timeout: 300
`

	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write config content: %v", err)
	}
	tmpFile.Close()

	adapter := NewConfigAdapter()
	if err := adapter.LoadConfig(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test GetVMConfig
	vm, err := adapter.GetVMConfig("test-vm")
	if err != nil {
		t.Fatalf("Failed to get VM config: %v", err)
	}

	if vm.Name != "test-vm" {
		t.Errorf("Expected VM name 'test-vm', got '%s'", vm.Name)
	}

	if vm.ID != 100 {
		t.Errorf("Expected VM ID 100, got %d", vm.ID)
	}

	// Test GetAllVMConfigs
	vms, err := adapter.GetAllVMConfigs()
	if err != nil {
		t.Fatalf("Failed to get all VM configs: %v", err)
	}

	if len(vms) != 1 {
		t.Errorf("Expected 1 VM, got %d", len(vms))
	}
}

func TestConfigAdapter_ValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: `
proxmox:
  host: "test-host"
  port: 8006
  user: "test@pam"
  password: "test-password"
  node: "test-node"
ssh:
  user: "root"
  password: "ssh-password"
  port: 22
  copy_local_key: false
vms: []
`,
			expectError: false,
		},
		{
			name: "missing proxmox host",
			config: `
proxmox:
  host: ""
  port: 8006
  user: "test@pam"
  password: "test-password"
  node: "test-node"
ssh:
  user: "root"
  password: "ssh-password"
  port: 22
  copy_local_key: false
vms: []
`,
			expectError: true,
			errorMsg:    "proxmox host is required",
		},
		{
			name: "invalid VM config",
			config: `
proxmox:
  host: "test-host"
  port: 8006
  user: "test@pam"
  password: "test-password"
  node: "test-node"
ssh:
  user: "root"
  password: "ssh-password"
  port: 22
  copy_local_key: false
vms:
  - name: ""
    vmid: 100
    cores: 2
    memory: 2048
    disk_size: "20G"
    network:
      bridge: "vmbr0"
      vlan: 100
      model: "virtio"
    os_template: "local:vztmpl/ubuntu-22.04-standard_22.04-1_amd64.tar.zst"
    tags: ["test"]
    auto_start: true
    ssh:
      user: "ubuntu"
      password: "ubuntu-password"
      key_path: ""
      copy_local_key: true
      authorized_keys: []
    scripts: []
`,
			expectError: true,
			errorMsg:    "name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.Write([]byte(tt.config)); err != nil {
				t.Fatalf("Failed to write config content: %v", err)
			}
			tmpFile.Close()

			adapter := NewConfigAdapter()
			if err := adapter.LoadConfig(tmpFile.Name()); err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			err = adapter.ValidateConfig()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
