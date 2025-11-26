package ports

import "proxima/internal/core/domain"

type VMService interface {
	CreateVM(vm *domain.VM) error
	StartVM(id int) error
	StopVM(id int) error
	ShutdownVM(id int) error
	DeleteVM(id int) error
	GetVM(id int) (*domain.VM, error)
	ListVMs() ([]*domain.VM, error)
	ExecuteScriptOnVM(vmid int, script *domain.Script) (*domain.Command, error)
	ExecuteCommandOnVM(vmid int, command string, args []string, timeout int) (*domain.Command, error)
	CopySSHKey(vmid int) error
}

type ConfigService interface {
	LoadConfig(path string) error
	GetVMConfig(name string) (*domain.VM, error)
	GetAllVMConfigs() ([]*domain.VM, error)
}
