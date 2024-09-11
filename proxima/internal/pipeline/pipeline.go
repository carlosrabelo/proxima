package pipeline

import (
	"context"
	"fmt"
	"os"

	"proxima/proxima/internal/config"
	"proxima/proxima/internal/diskbuild"
	"proxima/proxima/internal/proxmox"
	"proxima/proxima/internal/ui"
	"proxima/proxima/internal/upload"
)

// Runner executes the create pipeline stages in order.
type Runner struct {
	Reporter  ui.Reporter
	Build     func(ctx context.Context, cfg config.Config) (diskbuild.Result, error)
	Upload    func(ctx context.Context, cfg config.Config, local string) (upload.Result, error)
	Provision func(ctx context.Context, cfg config.Config, remote string) (proxmox.ProvisionResult, error)
	Start     func(ctx context.Context, cfg config.Config, vmid int) error
	WaitIP    func(ctx context.Context, cfg config.Config, vmid int) (string, error)
	KeepQcow2 bool
}

// CreateResult is the outcome of a successful create pipeline.
type CreateResult struct {
	VMID      int
	Name      string
	Qcow2Path string
	IP        string
}

// Create runs Validate → BuildLocal → Upload → Provision → Start → WaitIP.
func (r *Runner) Create(ctx context.Context, cfg config.Config) (CreateResult, error) {
	rep := r.Reporter
	if rep == nil {
		rep = &ui.TextReporter{Out: os.Stderr}
	}

	rep.StageStart("Validate")
	if err := config.Validate(cfg); err != nil {
		rep.StageFail("Validate", err)
		return CreateResult{}, err
	}
	rep.StageDone("Validate", "ok")

	rep.StageStart("BuildLocal")
	buildRes, err := r.Build(ctx, cfg)
	if err != nil {
		rep.StageFail("BuildLocal", err)
		return CreateResult{}, err
	}
	rep.StageDone("BuildLocal", fmt.Sprintf("%s, %d bytes", buildRes.Qcow2Path, buildRes.Bytes))

	rep.StageStart("Upload")
	upRes, err := r.Upload(ctx, cfg, buildRes.Qcow2Path)
	if err != nil {
		rep.StageFail("Upload", err)
		return CreateResult{}, err
	}
	rep.StageDone("Upload", fmt.Sprintf("%s, %d bytes", upRes.RemotePath, upRes.Bytes))

	rep.StageStart("Provision")
	provRes, err := r.Provision(ctx, cfg, upRes.RemotePath)
	if err != nil {
		rep.StageFail("Provision", err)
		return CreateResult{}, err
	}
	rep.StageDone("Provision", fmt.Sprintf("vmid=%d name=%s", provRes.VMID, provRes.Name))

	started := false
	if cfg.StartAfterCreate {
		rep.StageStart("Start")
		if err := r.Start(ctx, cfg, provRes.VMID); err != nil {
			rep.StageFail("Start", err)
			return CreateResult{}, err
		}
		rep.StageDone("Start", fmt.Sprintf("vmid=%d", provRes.VMID))
		started = true
	}

	ip := ""
	if cfg.WaitIP {
		if !started {
			err := fmt.Errorf("wait_ip requires a running vm: enable --start or start the vm first")
			rep.StageStart("WaitIP")
			rep.StageFail("WaitIP", err)
			return CreateResult{}, err
		}
		rep.StageStart("WaitIP")
		ip, err = r.WaitIP(ctx, cfg, provRes.VMID)
		if err != nil {
			rep.StageFail("WaitIP", err)
			return CreateResult{}, err
		}
		rep.StageDone("WaitIP", ip)
	}

	if !r.KeepQcow2 {
		_ = os.Remove(buildRes.Qcow2Path)
	}

	return CreateResult{
		VMID:      provRes.VMID,
		Name:      provRes.Name,
		Qcow2Path: buildRes.Qcow2Path,
		IP:        ip,
	}, nil
}
