package config

import (
	"fmt"
	"os"
	"proxima/internal/core/domain"
	"proxima/internal/core/ports"

	"gopkg.in/yaml.v3"
)

type ConfigAdapter struct {
	config *Config
}

func NewConfigAdapter() *ConfigAdapter {
	return &ConfigAdapter{}
}

func (c *ConfigAdapter) LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	c.config = &config
	return nil
}

func (c *ConfigAdapter) GetVMConfig(name string) (*domain.VM, error) {
	if c.config == nil {
		return nil, fmt.Errorf("config not loaded")
	}

	for _, vmConfig := range c.config.VMs {
		if vmConfig.Name == name {
			return c.convertToDomainVM(&vmConfig), nil
		}
	}

	return nil, fmt.Errorf("VM config with name %s not found", name)
}

func (c *ConfigAdapter) GetAllVMConfigs() ([]*domain.VM, error) {
	if c.config == nil {
		return nil, fmt.Errorf("config not loaded")
	}

	var vms []*domain.VM
	for _, vmConfig := range c.config.VMs {
		vm := c.convertToDomainVM(&vmConfig)
		vms = append(vms, vm)
	}

	return vms, nil
}

func (c *ConfigAdapter) GetProxmoxConfig() ProxmoxConfig {
	if c.config == nil {
		return ProxmoxConfig{}
	}
	return c.config.Proxmox
}

func (c *ConfigAdapter) GetSSHConfig() SSHConfig {
	if c.config == nil {
		return SSHConfig{}
	}
	return c.config.SSH
}

func (c *ConfigAdapter) ValidateConfig() error {
	if c.config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Validate Proxmox configuration
	if c.config.Proxmox.Host == "" {
		return fmt.Errorf("proxmox host is required")
	}
	if c.config.Proxmox.User == "" {
		return fmt.Errorf("proxmox user is required")
	}
	if c.config.Proxmox.Password == "" {
		return fmt.Errorf("proxmox password is required")
	}
	if c.config.Proxmox.Node == "" {
		return fmt.Errorf("proxmox node is required")
	}

	// Validate VM configurations
	for i, vm := range c.config.VMs {
		if vm.Name == "" {
			return fmt.Errorf("VM[%d]: name is required", i)
		}
		if vm.VMID <= 0 {
			return fmt.Errorf("VM[%d]: vmid must be positive", i)
		}

		// Check effective values (VM specific or Default)
		cores := vm.Cores
		if cores == 0 {
			cores = c.config.Defaults.Cores
		}
		if cores <= 0 {
			return fmt.Errorf("VM[%d]: cores must be positive (checked VM and Defaults)", i)
		}

		memory := vm.Memory
		if memory == 0 {
			memory = c.config.Defaults.Memory
		}
		if memory <= 0 {
			return fmt.Errorf("VM[%d]: memory must be positive (checked VM and Defaults)", i)
		}

		diskSize := vm.DiskSize
		if diskSize == "" {
			diskSize = c.config.Defaults.DiskSize
		}
		if diskSize == "" {
			return fmt.Errorf("VM[%d]: disk_size is required (checked VM and Defaults)", i)
		}

		// Validate that Template is provided (or aliased)
		template := vm.Template
		if template == "" {
			template = c.config.Defaults.Template
		}

		if template == "" {
			return fmt.Errorf("VM[%d]: template is required (checked VM and Defaults)", i)
		}
	}

	return nil
}

func (c *ConfigAdapter) convertToDomainVM(vmConfig *VMConfig) *domain.VM {
	vm := domain.NewVM(vmConfig.Name, vmConfig.VMID)

	// Merge Defaults
	defaults := c.config.Defaults

	vm.Cores = vmConfig.Cores
	if vm.Cores == 0 {
		vm.Cores = defaults.Cores
	}

	vm.Memory = vmConfig.Memory
	if vm.Memory == 0 {
		vm.Memory = defaults.Memory
	}

	vm.DiskSize = vmConfig.DiskSize
	if vm.DiskSize == "" {
		vm.DiskSize = defaults.DiskSize
	}

	// Network
	vm.Network = domain.Network{
		Bridge: vmConfig.Network.Bridge,
		VLAN:   vmConfig.Network.VLAN,
		Model:  vmConfig.Network.Model,
	}
	if vm.Network.Bridge == "" {
		vm.Network.Bridge = defaults.Network.Bridge
	}
	if vm.Network.Model == "" {
		vm.Network.Model = defaults.Network.Model
	}
	if vm.Network.VLAN == 0 {
		vm.Network.VLAN = defaults.Network.VLAN
	}

	// Template with Alias Resolution
	vm.Template = vmConfig.Template
	if vm.Template == "" {
		vm.Template = defaults.Template
	}
	// Try resolve alias
	if val, ok := c.config.Templates[vm.Template]; ok {
		vm.Template = val
	}

	vm.Tags = vmConfig.Tags
	if len(vm.Tags) == 0 {
		vm.Tags = defaults.Tags
	}

	vm.AutoStart = vmConfig.AutoStart
	// Boolean defaults are tricky, assuming false as valid default if not set manually,
	// but if we wanted true default we'd need *bool. For now, sticking to value override if false.

	// SSH
	vm.SSH = domain.SSHConfig{
		User:           vmConfig.SSH.User,
		Password:       vmConfig.SSH.Password,
		KeyPath:        vmConfig.SSH.KeyPath,
		AuthorizedKeys: vmConfig.SSH.AuthorizedKeys,
		CopyLocalKey:   vmConfig.SSH.CopyLocalKey,
	}
	// Basic string merge for SSH User
	if vm.SSH.User == "" {
		vm.SSH.User = defaults.SSH.User
	}
	// Note: SSH config normally falls back to global SSH config if empty, handled in InitializeServices usually,
	// but here we can merge VM defaults if provided.

	for _, scriptConfig := range vmConfig.Scripts {
		script := domain.Script{
			Name:    scriptConfig.Name,
			Path:    scriptConfig.Path,
			Args:    scriptConfig.Args,
			Timeout: scriptConfig.Timeout,
		}
		vm.Scripts = append(vm.Scripts, script)
	}

	return vm
}

var _ ports.ConfigService = (*ConfigAdapter)(nil)
