package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"proxima/proxima/internal/config"
	"proxima/proxima/internal/diskbuild"
	"proxima/proxima/internal/pipeline"
	"proxima/proxima/internal/proxmox"
	"proxima/proxima/internal/ui"
	"proxima/proxima/internal/upload"
)

func newCreateCommand() *cobra.Command {
	f := &configFlags{}
	startAfter := true
	waitIP := true
	keepQcow2 := true
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Build, upload, provision, start, and wait for guest IP",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, f, startAfter, waitIP, keepQcow2)
		},
	}
	bindConfigFlags(cmd, f)
	cmd.Flags().BoolVar(&startAfter, "start", true, "start VM after provision")
	cmd.Flags().BoolVar(&waitIP, "wait-ip", true, "wait for guest IPv4 after start")
	cmd.Flags().BoolVar(&keepQcow2, "keep-qcow2", true, "keep local qcow2 after success")
	return cmd
}

func runCreate(cmd *cobra.Command, f *configFlags, startAfter, waitIP, keepQcow2 bool) error {
	path, err := resolveConfigPath(f.configPath)
	if err != nil {
		return err
	}

	o := overrideFromFlags(cmd, f)
	if cmd.Flags().Changed("start") {
		o.StartAfterCreate = &startAfter
	}
	if cmd.Flags().Changed("wait-ip") {
		o.WaitIP = &waitIP
	}
	cfg, err := config.Load(path, o)
	if err != nil {
		return err
	}
	if cmd.Flags().Changed("start") {
		cfg.StartAfterCreate = startAfter
	}
	if cmd.Flags().Changed("wait-ip") {
		cfg.WaitIP = waitIP
	}

	runner := &pipeline.Runner{
		Reporter:  &ui.TextReporter{Out: cmd.ErrOrStderr()},
		KeepQcow2: keepQcow2,
		Build: func(ctx context.Context, c config.Config) (diskbuild.Result, error) {
			b := &diskbuild.Builder{Docker: buildDockerFactory()}
			return b.Build(ctx, c)
		},
		Upload: func(ctx context.Context, c config.Config, local string) (upload.Result, error) {
			u, err := uploadFactory(c)
			if err != nil {
				return upload.Result{}, err
			}
			return u.UploadQcow2(ctx, local, c.VMID)
		},
		Provision: func(ctx context.Context, c config.Config, remote string) (proxmox.ProvisionResult, error) {
			client, err := provisionClientFactory(c)
			if err != nil {
				return proxmox.ProvisionResult{}, err
			}
			return client.ProvisionVM(ctx, proxmox.ProvisionRequest{Config: c, RemoteDisk: remote})
		},
		Start: func(ctx context.Context, c config.Config, vmid int) error {
			client, err := startClientFactory(c)
			if err != nil {
				return err
			}
			return client.StartVM(ctx, vmid)
		},
		WaitIP: func(ctx context.Context, c config.Config, vmid int) (string, error) {
			client, err := ipClientFactory(c)
			if err != nil {
				return "", err
			}
			return client.WaitGuestIPv4(ctx, vmid, c.AgentWaitTimeout, c.AgentPollInterval)
		},
	}

	res, err := runner.Create(context.Background(), cfg)
	if err != nil {
		return mapCreateError(err)
	}

	if res.IP != "" {
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "created vmid=%d name=%s ip=%s\n", res.VMID, res.Name, res.IP)
	} else {
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "created vmid=%d name=%s\n", res.VMID, res.Name)
	}
	return err
}

func mapCreateError(err error) error {
	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, "local disk build failed:"):
		return newExitError(2, msg)
	case strings.HasPrefix(msg, "vm provision failed and cleanup failed:"):
		return newExitError(4, msg)
	case strings.HasPrefix(msg, "disk upload failed:"),
		strings.HasPrefix(msg, "vm provision failed:"),
		strings.HasPrefix(msg, "vm start failed:"),
		strings.HasPrefix(msg, "guest ip wait timed out"),
		strings.HasPrefix(msg, "proxmox "):
		return newExitError(3, msg)
	default:
		return err
	}
}
