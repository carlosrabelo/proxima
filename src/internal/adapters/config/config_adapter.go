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

func (c *ConfigAdapter) convertToDomainVM(vmConfig *VMConfig) *domain.VM {
	vm := domain.NewVM(vmConfig.Name, vmConfig.VMID)
	vm.Cores = vmConfig.Cores
	vm.Memory = vmConfig.Memory
	vm.DiskSize = vmConfig.DiskSize
	vm.Network = domain.Network{
		Bridge: vmConfig.Network.Bridge,
		VLAN:   vmConfig.Network.VLAN,
		Model:  vmConfig.Network.Model,
	}
	vm.OSTemplate = vmConfig.OSTemplate
	vm.Tags = vmConfig.Tags
	vm.AutoStart = vmConfig.AutoStart
	vm.SSH = domain.SSHConfig{
		User:           vmConfig.SSH.User,
		Password:       vmConfig.SSH.Password,
		KeyPath:        vmConfig.SSH.KeyPath,
		AuthorizedKeys: vmConfig.SSH.AuthorizedKeys,
		CopyLocalKey:   vmConfig.SSH.CopyLocalKey,
	}

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
		if vm.Cores <= 0 {
			return fmt.Errorf("VM[%d]: cores must be positive", i)
		}
		if vm.Memory <= 0 {
			return fmt.Errorf("VM[%d]: memory must be positive", i)
		}
		if vm.DiskSize == "" {
			return fmt.Errorf("VM[%d]: disk_size is required", i)
		}
		if vm.OSTemplate == "" {
			return fmt.Errorf("VM[%d]: os_template is required", i)
		}

		// Validate scripts if any
		for j, script := range vm.Scripts {
			if script.Name == "" {
				return fmt.Errorf("VM[%d].Script[%d]: name is required", i, j)
			}
			if script.Path == "" {
				return fmt.Errorf("VM[%d].Script[%d]: path is required", i, j)
			}
			if script.Timeout <= 0 {
				return fmt.Errorf("VM[%d].Script[%d]: timeout must be positive", i, j)
			}
		}
	}

	return nil
}

var _ ports.ConfigService = (*ConfigAdapter)(nil)
