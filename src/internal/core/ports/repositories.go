package ports

import "proxima/internal/core/domain"

type VMRepository interface {
	Create(vm *domain.VM) error
	GetByID(id int) (*domain.VM, error)
	GetByName(name string) (*domain.VM, error)
	Update(vm *domain.VM) error
	Delete(id int) error
	List() ([]*domain.VM, error)
	Start(id int) error
	Stop(id int) error
	Shutdown(id int) error
	GetStatus(id int) (domain.VMStatus, error)
}

type SSHRepository interface {
	ExecuteCommand(vmid int, command string, args []string, timeout int) (*domain.Command, error)
	ExecuteScript(vmid int, scriptPath string, args []string, timeout int) (*domain.Command, error)
	GetCommandHistory(vmid int) ([]*domain.Command, error)
	CopyLocalPublicKey(vmid int) error
}
