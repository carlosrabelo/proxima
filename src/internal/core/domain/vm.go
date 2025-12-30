package domain

import "time"

type VM struct {
	ID        int
	Name      string
	Status    VMStatus
	Cores     int
	Memory    int
	DiskSize  string
	Network   Network
	Template  string
	Tags      []string
	AutoStart bool
	SSH       SSHConfig
	Scripts   []Script
	CreatedAt time.Time
	UpdatedAt time.Time
}

type VMStatus string

const (
	VMStatusRunning  VMStatus = "running"
	VMStatusStopped  VMStatus = "stopped"
	VMStatusStarting VMStatus = "starting"
	VMStatusCreating VMStatus = "creating"
	VMStatusDeleting VMStatus = "deleting"
	VMStatusError    VMStatus = "error"
)

type Network struct {
	Bridge string
	VLAN   int
	Model  string
}

type SSHConfig struct {
	User           string
	Password       string
	KeyPath        string
	AuthorizedKeys []string
	CopyLocalKey   bool
}

type Script struct {
	Name    string
	Path    string
	Args    []string
	Timeout int
}

func NewVM(name string, vmid int) *VM {
	return &VM{
		ID:        vmid,
		Name:      name,
		Status:    VMStatusStopped,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (vm *VM) Start() {
	vm.Status = VMStatusRunning
	vm.UpdatedAt = time.Now()
}

func (vm *VM) Stop() {
	vm.Status = VMStatusStopped
	vm.UpdatedAt = time.Now()
}

func (vm *VM) Delete() {
	vm.Status = VMStatusDeleting
	vm.UpdatedAt = time.Now()
}
