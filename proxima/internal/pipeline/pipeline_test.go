package pipeline_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"proxima/proxima/internal/config"
	"proxima/proxima/internal/diskbuild"
	"proxima/proxima/internal/pipeline"
	"proxima/proxima/internal/proxmox"
	"proxima/proxima/internal/ui"
	"proxima/proxima/internal/upload"
)

func validCfg() config.Config {
	return config.Config{
		ProxmoxHost:       "192.168.1.10",
		Node:              "pve",
		Auth:              config.Auth{User: "root@pam", Password: "secret"},
		VMID:              200,
		VMName:            "alpine-docker",
		MemoryMiB:         2048,
		Cores:             2,
		DiskSize:          "8G",
		RootPassword:      "change-me",
		StartAfterCreate:  true,
		WaitIP:            true,
		AgentWaitTimeout:  120 * time.Second,
		AgentPollInterval: 2 * time.Second,
	}
}

func TestCreateHappyPathStageOrder(t *testing.T) {
	var buf bytes.Buffer
	var stages []string
	r := &pipeline.Runner{
		Reporter:  &ui.TextReporter{Out: &buf},
		KeepQcow2: true,
		Build: func(ctx context.Context, cfg config.Config) (diskbuild.Result, error) {
			stages = append(stages, "BuildLocal")
			return diskbuild.Result{Qcow2Path: "alpine-ready.qcow2", Bytes: 10}, nil
		},
		Upload: func(ctx context.Context, cfg config.Config, local string) (upload.Result, error) {
			stages = append(stages, "Upload")
			return upload.Result{LocalPath: local, RemotePath: "/var/tmp/proxima/200-disk.qcow2", Bytes: 10}, nil
		},
		Provision: func(ctx context.Context, cfg config.Config, remote string) (proxmox.ProvisionResult, error) {
			stages = append(stages, "Provision")
			return proxmox.ProvisionResult{VMID: 200, Name: "alpine-docker"}, nil
		},
		Start: func(ctx context.Context, cfg config.Config, vmid int) error {
			stages = append(stages, "Start")
			return nil
		},
		WaitIP: func(ctx context.Context, cfg config.Config, vmid int) (string, error) {
			stages = append(stages, "WaitIP")
			return "10.0.0.5", nil
		},
	}

	res, err := r.Create(context.Background(), validCfg())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.IP != "10.0.0.5" || res.VMID != 200 {
		t.Fatalf("result = %+v", res)
	}
	wantOrder := []string{"BuildLocal", "Upload", "Provision", "Start", "WaitIP"}
	if strings.Join(stages, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("stages = %v, want %v", stages, wantOrder)
	}
	out := buf.String()
	for _, line := range []string{
		"==> Validate: starting",
		"==> Validate: done (ok)",
		"==> BuildLocal: starting",
		"==> BuildLocal: done (alpine-ready.qcow2, 10 bytes)",
		"==> Upload: starting",
		"==> Upload: done (/var/tmp/proxima/200-disk.qcow2, 10 bytes)",
		"==> Provision: starting",
		"==> Provision: done (vmid=200 name=alpine-docker)",
		"==> Start: starting",
		"==> Start: done (vmid=200)",
		"==> WaitIP: starting",
		"==> WaitIP: done (10.0.0.5)",
	} {
		if !strings.Contains(out, line) {
			t.Fatalf("missing %q in %q", line, out)
		}
	}
}

func TestCreateStopsOnBuildFailure(t *testing.T) {
	calledUpload := false
	r := &pipeline.Runner{
		Reporter:  &ui.TextReporter{Out: &bytes.Buffer{}},
		KeepQcow2: true,
		Build: func(ctx context.Context, cfg config.Config) (diskbuild.Result, error) {
			return diskbuild.Result{}, errors.New("local disk build failed: boom")
		},
		Upload: func(ctx context.Context, cfg config.Config, local string) (upload.Result, error) {
			calledUpload = true
			return upload.Result{}, nil
		},
	}
	_, err := r.Create(context.Background(), validCfg())
	if err == nil {
		t.Fatal("expected build failure")
	}
	if calledUpload {
		t.Fatal("upload must not run after build failure")
	}
}

func TestCreateStopsOnUploadFailure(t *testing.T) {
	calledProvision := false
	r := &pipeline.Runner{
		Reporter:  &ui.TextReporter{Out: &bytes.Buffer{}},
		KeepQcow2: true,
		Build: func(ctx context.Context, cfg config.Config) (diskbuild.Result, error) {
			return diskbuild.Result{Qcow2Path: "alpine-ready.qcow2", Bytes: 1}, nil
		},
		Upload: func(ctx context.Context, cfg config.Config, local string) (upload.Result, error) {
			return upload.Result{}, errors.New("disk upload failed: boom")
		},
		Provision: func(ctx context.Context, cfg config.Config, remote string) (proxmox.ProvisionResult, error) {
			calledProvision = true
			return proxmox.ProvisionResult{}, nil
		},
	}
	_, err := r.Create(context.Background(), validCfg())
	if err == nil {
		t.Fatal("expected upload failure")
	}
	if calledProvision {
		t.Fatal("provision must not run after upload failure")
	}
}
