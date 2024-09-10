package proxmox_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"proxima/proxima/internal/proxmox"
)

func TestStartVMIdempotentWhenRunning(t *testing.T) {
	fake := &fakeSSH{
		responses: map[string]string{
			"qm status 200": "status: running\n",
		},
	}
	client := &proxmox.Client{}
	client.SetSSHOpenForTest(func(ctx context.Context) (proxmox.SSHSession, error) {
		return fake, nil
	})

	if err := client.StartVM(context.Background(), 200); err != nil {
		t.Fatalf("StartVM: %v", err)
	}
	for _, cmd := range fake.commands {
		if strings.HasPrefix(cmd, "qm start ") {
			t.Fatalf("must not qm start when already running: %#v", fake.commands)
		}
	}
}

func TestStartVMErrorWrapped(t *testing.T) {
	fake := &fakeSSH{
		responses: map[string]string{
			"qm status 200": "status: stopped\n",
		},
		errors: map[string]error{
			"qm start 200": fmt.Errorf("node offline"),
		},
	}
	client := &proxmox.Client{}
	client.SetSSHOpenForTest(func(ctx context.Context) (proxmox.SSHSession, error) {
		return fake, nil
	})

	err := client.StartVM(context.Background(), 200)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.HasPrefix(err.Error(), "vm start failed:") {
		t.Fatalf("error = %q", err.Error())
	}
	if !strings.Contains(err.Error(), "node offline") {
		t.Fatalf("error = %q", err.Error())
	}
}
