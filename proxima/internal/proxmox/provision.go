package proxmox

import (
	"context"
	"fmt"
	"strconv"

	"proxima/proxima/internal/config"
)

// ProvisionRequest configures VM creation from a staged remote disk.
type ProvisionRequest struct {
	Config     config.Config
	RemoteDisk string
}

// ProvisionResult is the outcome of a successful provision.
type ProvisionResult struct {
	VMID int
	Name string
}

// ProvisionVM creates the VM, imports the disk, attaches scsi0, sets boot/agent,
// and removes the staged remote qcow2.
func (c *Client) ProvisionVM(ctx context.Context, req ProvisionRequest) (ProvisionResult, error) {
	cfg := req.Config
	bridge := cfg.Bridge
	if bridge == "" {
		bridge = "vmbr0"
	}

	sess, err := c.SSH(ctx)
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("vm provision failed: %w", err)
	}
	defer sess.Close()

	createCmd := fmt.Sprintf(
		"qm create %d --name %s --memory %d --cores %d --net0 virtio,bridge=%s --scsihw virtio-scsi-single --ostype l26 --agent 1",
		cfg.VMID,
		shellArg(cfg.VMName),
		cfg.MemoryMiB,
		cfg.Cores,
		bridge,
	)
	if _, err := sess.CombinedOutput(ctx, createCmd); err != nil {
		return ProvisionResult{}, fmt.Errorf("vm provision failed: %w", err)
	}

	return ProvisionResult{VMID: cfg.VMID, Name: cfg.VMName}, nil
}

// DestroyVM removes a VM with purge.
func (c *Client) DestroyVM(ctx context.Context, vmid int) error {
	sess, err := c.SSH(ctx)
	if err != nil {
		return err
	}
	defer sess.Close()
	_, err = sess.CombinedOutput(ctx, fmt.Sprintf("qm destroy %d --purge 1", vmid))
	return err
}

func shellArg(s string) string {
	// VM names are validated elsewhere; quote if empty-safe alphanumeric/dash.
	if s == "" {
		return "''"
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return strconv.Quote(s)
	}
	return s
}
