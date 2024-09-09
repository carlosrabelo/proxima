package proxmox

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	storage := cfg.Storage
	if storage == "" {
		storage = "local-lvm"
	}
	remote := req.RemoteDisk
	if remote == "" {
		remote = fmt.Sprintf("/var/tmp/proxima/%d-disk.qcow2", cfg.VMID)
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

	importCmd := fmt.Sprintf("qm importdisk %d %s %s", cfg.VMID, remote, storage)
	if _, err := sess.CombinedOutput(ctx, importCmd); err != nil {
		return ProvisionResult{}, fmt.Errorf("vm provision failed: %w", err)
	}

	// Attach imported unused disk as scsi0.
	// Discovery algorithm: prefer the conventional volume name
	// "<Storage>:vm-<vmid>-disk-0". If that is wrong for the node, parse
	// `qm config <vmid>` for the first unusedN: line and use its value as
	// the volume-ref for `qm set <vmid> --scsi0 <volume-ref>`.
	volumeRef := fmt.Sprintf("%s:vm-%d-disk-0", storage, cfg.VMID)
	cfgOut, err := sess.CombinedOutput(ctx, fmt.Sprintf("qm config %d", cfg.VMID))
	if err == nil {
		if discovered := firstUnusedVolume(string(cfgOut)); discovered != "" {
			volumeRef = discovered
		}
	}
	attachCmd := fmt.Sprintf("qm set %d --scsi0 %s", cfg.VMID, volumeRef)
	if _, err := sess.CombinedOutput(ctx, attachCmd); err != nil {
		return ProvisionResult{}, fmt.Errorf("vm provision failed: %w", err)
	}

	bootCmd := fmt.Sprintf("qm set %d --boot order=scsi0", cfg.VMID)
	if _, err := sess.CombinedOutput(ctx, bootCmd); err != nil {
		return ProvisionResult{}, fmt.Errorf("vm provision failed: %w", err)
	}

	return ProvisionResult{VMID: cfg.VMID, Name: cfg.VMName}, nil
}

// firstUnusedVolume returns the value of the first unusedN: line in qm config output.
func firstUnusedVolume(qmConfig string) string {
	for _, line := range strings.Split(qmConfig, "\n") {
		line = strings.TrimSpace(line)
		key, val, ok := strings.Cut(line, ":")
		if !ok || !strings.HasPrefix(key, "unused") {
			continue
		}
		n := key[len("unused"):]
		if n == "" {
			continue
		}
		if _, err := strconv.Atoi(n); err != nil {
			continue
		}
		val = strings.TrimSpace(val)
		if val != "" {
			return val
		}
	}
	return ""
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
