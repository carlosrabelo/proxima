package proxmox_ssh

import (
	"bufio"
	"fmt"
	"os/exec"
	"proxima/internal/core/domain"
	"proxima/internal/core/ports"
	"strconv"
	"strings"
)

type ProxmoxSSHAdapter struct {
	host     string
	user     string
	password string
	port     int
	vmIPBase string
}

func NewProxmoxSSHAdapter(host, user, password string, port int) *ProxmoxSSHAdapter {
	// Extract VM IP base from host
	vmIPBase := strings.Join(strings.Split(host, ".")[:3], ".")

	return &ProxmoxSSHAdapter{
		host:     host,
		user:     user,
		password: password,
		port:     port,
		vmIPBase: vmIPBase,
	}
}

func (p *ProxmoxSSHAdapter) executeSSHCommand(command string) (string, error) {
	var cmd *exec.Cmd

	if p.password != "" {
		// Use sshpass for password authentication
		cmd = exec.Command("sshpass", "-p", p.password, "ssh", "-o", "StrictHostKeyChecking=no", "-p", strconv.Itoa(p.port), fmt.Sprintf("%s@%s", p.user, p.host), command)
	} else {
		// Use SSH key authentication
		cmd = exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-p", strconv.Itoa(p.port), fmt.Sprintf("%s@%s", p.user, p.host), command)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("SSH command failed: %w, output: %s", err, string(output))
	}

	return string(output), nil
}

func (p *ProxmoxSSHAdapter) Create(vm *domain.VM) error {
	// Build qm create command
	cmd := fmt.Sprintf("qm create %d --name %s --cores %d --memory %d --net0 virtio,bridge=%s",
		vm.ID, vm.Name, vm.Cores, vm.Memory, vm.Network.Bridge)

	if vm.Network.VLAN > 0 {
		cmd += fmt.Sprintf(",tag=%d", vm.Network.VLAN)
	}

	if vm.DiskSize != "" {
		// If OSTemplate is provided, we might want to use it as the source for the disk
		// But for now, let's keep disk creation simple and handle OSTemplate as cdrom
		cmd += fmt.Sprintf(" --scsi0 %s", vm.DiskSize)
	}

	if vm.OSTemplate != "" {
		cmd += fmt.Sprintf(" --cdrom %s", vm.OSTemplate)
	}

	// Add basic cloud-init setup if keys are present (requires cloud-init drive, which we assume or create)
	// For now, we'll just append the command. In a real scenario, we'd need to add a cloud-init drive.
	// cmd += " --ide2 local-lvm:cloudinit"

	fmt.Printf("Creating VM with command: %s\n", cmd)
	_, err := p.executeSSHCommand(cmd)
	if err != nil {
		return err
	}

	if len(vm.SSH.AuthorizedKeys) > 0 {
		fmt.Printf("[WARNING] SSH keys defined. Ensure Cloud-Init is configured or use 'proxima copy-key' after boot.\n")
	}

	return nil
}

func (p *ProxmoxSSHAdapter) GetByID(id int) (*domain.VM, error) {
	output, err := p.executeSSHCommand(fmt.Sprintf("qm config %d", id))
	if err != nil {
		return nil, err
	}

	vm := domain.NewVM("", id)

	// Parse qm config output
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "name: ") {
			vm.Name = strings.TrimPrefix(line, "name: ")
		} else if strings.HasPrefix(line, "cores: ") {
			fmt.Sscanf(strings.TrimPrefix(line, "cores: "), "%d", &vm.Cores)
		} else if strings.HasPrefix(line, "memory: ") {
			fmt.Sscanf(strings.TrimPrefix(line, "memory: "), "%d", &vm.Memory)
		}
	}

	// Get status
	statusOutput, err := p.executeSSHCommand(fmt.Sprintf("qm status %d", id))
	if err == nil {
		if strings.Contains(statusOutput, "running") {
			vm.Status = domain.VMStatusRunning
		} else if strings.Contains(statusOutput, "stopped") {
			vm.Status = domain.VMStatusStopped
		}
	}

	return vm, nil
}

func (p *ProxmoxSSHAdapter) GetByName(name string) (*domain.VM, error) {
	vms, err := p.List()
	if err != nil {
		return nil, err
	}

	for _, vm := range vms {
		if vm.Name == name {
			return vm, nil
		}
	}

	return nil, fmt.Errorf("VM with name %s not found", name)
}

func (p *ProxmoxSSHAdapter) Update(vm *domain.VM) error {
	return fmt.Errorf("update operation not implemented for SSH adapter")
}

func (p *ProxmoxSSHAdapter) Delete(id int) error {
	_, err := p.executeSSHCommand(fmt.Sprintf("qm destroy %d", id))
	return err
}

func (p *ProxmoxSSHAdapter) List() ([]*domain.VM, error) {
	output, err := p.executeSSHCommand("qm list")
	if err != nil {
		return nil, err
	}

	var vms []*domain.VM
	scanner := bufio.NewScanner(strings.NewReader(output))

	// Skip header line
	if scanner.Scan() {
		// header
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			vmid, err := strconv.Atoi(fields[0])
			if err != nil {
				continue
			}

			name := fields[1]
			status := fields[2]

			vm := domain.NewVM(name, vmid)

			switch status {
			case "running":
				vm.Status = domain.VMStatusRunning
			case "stopped":
				vm.Status = domain.VMStatusStopped
			default:
				vm.Status = domain.VMStatus(status)
			}

			// Get detailed info (CPU, memory) using qm config
			if err := p.getVMDetails(vm); err != nil {
				// Continue even if we can't get details
				continue
			}

			vms = append(vms, vm)
		}
	}

	return vms, nil
}

func (p *ProxmoxSSHAdapter) getVMDetails(vm *domain.VM) error {
	output, err := p.executeSSHCommand(fmt.Sprintf("qm config %d", vm.ID))
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "cores: ") {
			var cores int
			fmt.Sscanf(strings.TrimPrefix(line, "cores: "), "%d", &cores)
			vm.Cores = cores
		} else if strings.HasPrefix(line, "memory: ") {
			var memory int
			fmt.Sscanf(strings.TrimPrefix(line, "memory: "), "%d", &memory)
			vm.Memory = memory
		}
	}

	return nil
}

func (p *ProxmoxSSHAdapter) Start(id int) error {
	_, err := p.executeSSHCommand(fmt.Sprintf("qm start %d", id))
	return err
}

func (p *ProxmoxSSHAdapter) Stop(id int) error {
	_, err := p.executeSSHCommand(fmt.Sprintf("qm stop %d", id))
	return err
}

func (p *ProxmoxSSHAdapter) Shutdown(id int) error {
	_, err := p.executeSSHCommand(fmt.Sprintf("qm shutdown %d", id))
	return err
}

func (p *ProxmoxSSHAdapter) GetStatus(id int) (domain.VMStatus, error) {
	output, err := p.executeSSHCommand(fmt.Sprintf("qm status %d", id))
	if err != nil {
		return "", err
	}

	if strings.Contains(output, "running") {
		return domain.VMStatusRunning, nil
	} else if strings.Contains(output, "stopped") {
		return domain.VMStatusStopped, nil
	} else if strings.Contains(output, "starting") {
		return domain.VMStatusStarting, nil
	}

	return domain.VMStatus(output), nil
}

func (p *ProxmoxSSHAdapter) GetVMIP(id int) (string, error) {
	// Get IP from QEMU Agent - this is the only supported method
	output, err := p.executeSSHCommand(fmt.Sprintf("qm agent %d network-get-interfaces", id))
	if err != nil {
		return "", fmt.Errorf("QEMU Agent not available or failed to get IP for VM %d: %w. Please ensure QEMU Guest Agent is installed and running on the VM", id, err)
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "ip-address:") && strings.Contains(line, "ipv4") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "ip-address:" && i+1 < len(fields) {
					ip := fields[i+1]
					// Validate IP format (basic check)
					if strings.Count(ip, ".") == 3 && !strings.HasPrefix(ip, "127.") && !strings.HasPrefix(ip, "169.254.") {
						return ip, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no valid IPv4 address found for VM %d via QEMU Agent. Please ensure QEMU Guest Agent is installed and running, and the VM has a valid network configuration", id)
}

var _ ports.VMRepository = (*ProxmoxSSHAdapter)(nil)
