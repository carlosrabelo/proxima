package service

import (
	"fmt"
	"proxima/internal/core/domain"
	"proxima/internal/core/ports"
	"time"
)

type VMService struct {
	vmRepo  ports.VMRepository
	sshRepo ports.SSHRepository
}

func NewVMService(vmRepo ports.VMRepository, sshRepo ports.SSHRepository) ports.VMService {
	return &VMService{
		vmRepo:  vmRepo,
		sshRepo: sshRepo,
	}
}

func (s *VMService) CreateVM(vm *domain.VM) error {
	if err := s.vmRepo.Create(vm); err != nil {
		return fmt.Errorf("failed to create VM: %w", err)
	}

	if vm.AutoStart {
		return s.StartVM(vm.ID)
	}

	return nil
}

func (s *VMService) StartVM(id int) error {
	if err := s.vmRepo.Start(id); err != nil {
		return fmt.Errorf("failed to start VM %d: %w", id, err)
	}
	return nil
}

func (s *VMService) StopVM(id int) error {
	if err := s.vmRepo.Stop(id); err != nil {
		return fmt.Errorf("failed to stop VM %d: %w", id, err)
	}
	return nil
}

func (s *VMService) ShutdownVM(id int) error {
	// Check if VM is running first
	status, err := s.vmRepo.GetStatus(id)
	if err != nil {
		return fmt.Errorf("failed to get VM %d status: %w", id, err)
	}

	if status == domain.VMStatusStopped {
		fmt.Printf("VM %d is already stopped\n", id)
		return nil
	}

	// If VM is starting, wait for it to fully start before shutting down
	if status == domain.VMStatusStarting {
		fmt.Printf("VM %d is starting, waiting for it to fully start before shutdown...\n", id)
		for i := 0; i < 120; i++ { // Wait up to 2 minutes for startup
			time.Sleep(1 * time.Second)
			currentStatus, err := s.vmRepo.GetStatus(id)
			if err != nil {
				continue
			}
			if currentStatus == domain.VMStatusRunning {
				fmt.Printf("VM %d is now running, proceeding with shutdown...\n", id)
				break
			}
			if currentStatus == domain.VMStatusStopped {
				fmt.Printf("VM %d stopped during startup phase\n", id)
				return nil
			}
			if i == 119 { // Last iteration
				return fmt.Errorf("timeout waiting for VM %d to fully start", id)
			}
		}
	}

	if err := s.vmRepo.Shutdown(id); err != nil {
		return fmt.Errorf("failed to shutdown VM %d: %w", id, err)
	}

	// Wait for VM to actually shutdown (max 60 seconds)
	fmt.Printf("Waiting for VM %d to shutdown...\n", id)
	for i := 0; i < 60; i++ {
		time.Sleep(1 * time.Second)
		currentStatus, err := s.vmRepo.GetStatus(id)
		if err != nil {
			// If we can't get status during shutdown, that might be normal
			continue
		}
		if currentStatus == domain.VMStatusStopped {
			fmt.Printf("VM %d shutdown successfully\n", id)
			return nil
		}
	}

	return fmt.Errorf("timeout waiting for VM %d to shutdown", id)
}

func (s *VMService) DeleteVM(id int) error {
	if err := s.vmRepo.Stop(id); err != nil {
		return fmt.Errorf("failed to stop VM before deletion %d: %w", id, err)
	}

	if err := s.vmRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete VM %d: %w", id, err)
	}

	return nil
}

func (s *VMService) GetVM(id int) (*domain.VM, error) {
	vm, err := s.vmRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM %d: %w", id, err)
	}
	return vm, nil
}

func (s *VMService) ListVMs() ([]*domain.VM, error) {
	vms, err := s.vmRepo.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}
	return vms, nil
}

func (s *VMService) ExecuteScriptOnVM(vmid int, script *domain.Script) (*domain.Command, error) {
	command, err := s.sshRepo.ExecuteScript(vmid, script.Path, script.Args, script.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to execute script %s on VM %d: %w", script.Name, vmid, err)
	}
	return command, nil
}

func (s *VMService) ExecuteCommandOnVM(vmid int, command string, args []string, timeout int) (*domain.Command, error) {
	cmd, err := s.sshRepo.ExecuteCommand(vmid, command, args, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command on VM %d: %w", vmid, err)
	}
	return cmd, nil
}

func (s *VMService) CopySSHKey(vmid int) error {
	if err := s.sshRepo.CopyLocalPublicKey(vmid); err != nil {
		return fmt.Errorf("failed to copy SSH key to VM %d: %w", vmid, err)
	}
	return nil
}
