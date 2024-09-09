package proxmox_test

import (
	"context"
	"strings"
	"testing"

	"proxima/proxima/internal/config"
	"proxima/proxima/internal/proxmox"
)

type fakeSSH struct {
	commands  []string
	responses map[string]string
	errors    map[string]error
}

func (f *fakeSSH) CombinedOutput(ctx context.Context, command string) ([]byte, error) {
	_ = ctx
	f.commands = append(f.commands, command)
	if f.errors != nil {
		if err, ok := f.errors[command]; ok {
			return nil, err
		}
	}
	if f.responses != nil {
		if out, ok := f.responses[command]; ok {
			return []byte(out), nil
		}
	}
	return nil, nil
}

func (f *fakeSSH) Close() error { return nil }

func TestProvisionQMCreateArgs(t *testing.T) {
	fake := &fakeSSH{}
	client := &proxmox.Client{}
	client.SetSSHOpenForTest(func(ctx context.Context) (proxmox.SSHSession, error) {
		return fake, nil
	})

	cfg := config.Config{
		VMID:      200,
		VMName:    "alpine-docker",
		MemoryMiB: 2048,
		Cores:     2,
		Bridge:    "vmbr0",
		Storage:   "local-lvm",
	}
	_, err := client.ProvisionVM(context.Background(), proxmox.ProvisionRequest{
		Config:     cfg,
		RemoteDisk: "/var/tmp/proxima/200-disk.qcow2",
	})
	if err != nil {
		t.Fatalf("ProvisionVM: %v", err)
	}
	if len(fake.commands) == 0 {
		t.Fatal("no SSH commands recorded")
	}
	got := fake.commands[0]
	wantParts := []string{
		"qm create 200",
		"--name alpine-docker",
		"--memory 2048",
		"--cores 2",
		"--net0 virtio,bridge=vmbr0",
		"--scsihw virtio-scsi-single",
		"--ostype l26",
		"--agent 1",
	}
	for _, part := range wantParts {
		if !strings.Contains(got, part) {
			t.Fatalf("create cmd %q missing %q", got, part)
		}
	}
}

func TestProvisionImportAndAttach(t *testing.T) {
	fake := &fakeSSH{
		responses: map[string]string{
			"qm config 200": "unused0: local-lvm:vm-200-disk-0\n",
		},
	}
	client := &proxmox.Client{}
	client.SetSSHOpenForTest(func(ctx context.Context) (proxmox.SSHSession, error) {
		return fake, nil
	})

	_, err := client.ProvisionVM(context.Background(), proxmox.ProvisionRequest{
		Config: config.Config{
			VMID:      200,
			VMName:    "alpine-docker",
			MemoryMiB: 2048,
			Cores:     2,
			Bridge:    "vmbr0",
			Storage:   "local-lvm",
		},
		RemoteDisk: "/var/tmp/proxima/200-disk.qcow2",
	})
	if err != nil {
		t.Fatalf("ProvisionVM: %v", err)
	}

	var sawImport, sawAttach bool
	for _, cmd := range fake.commands {
		if cmd == "qm importdisk 200 /var/tmp/proxima/200-disk.qcow2 local-lvm" {
			sawImport = true
		}
		if cmd == "qm set 200 --scsi0 local-lvm:vm-200-disk-0" {
			sawAttach = true
		}
	}
	if !sawImport {
		t.Fatalf("missing importdisk in %#v", fake.commands)
	}
	if !sawAttach {
		t.Fatalf("missing scsi0 attach in %#v", fake.commands)
	}
}
