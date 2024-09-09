package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"proxima/proxima/internal/config"
	"proxima/proxima/internal/proxmox"
	"proxima/proxima/internal/upload"
)

var provisionClientFactory = defaultProvisionClientFactory

func defaultProvisionClientFactory(cfg config.Config) (*proxmox.Client, error) {
	return proxmox.NewClient(cfg)
}

// SetProvisionClientFactoryForTest replaces the Proxmox client factory (tests only).
func SetProvisionClientFactoryForTest(fn func(config.Config) (*proxmox.Client, error)) {
	provisionClientFactory = fn
}

// ResetProvisionClientFactoryForTest restores the default client factory.
func ResetProvisionClientFactoryForTest() {
	provisionClientFactory = defaultProvisionClientFactory
}

func newProvisionCommand() *cobra.Command {
	f := &configFlags{}
	var remoteDisk string
	cmd := &cobra.Command{
		Use:   "provision",
		Short: "Create and configure a Proxmox VM from a staged remote qcow2",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProvision(cmd, f, remoteDisk)
		},
	}
	bindConfigFlags(cmd, f)
	cmd.Flags().StringVar(&remoteDisk, "remote-disk", "", "remote staged qcow2 path")
	return cmd
}

func runProvision(cmd *cobra.Command, f *configFlags, remoteDisk string) error {
	if f.configPath == "" {
		return fmt.Errorf("config file must end in .yaml, .yml, or .json")
	}

	cfg, err := config.Load(f.configPath, overrideFromFlags(cmd, f))
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	if remoteDisk == "" {
		remoteDisk = upload.RemotePath(cfg.VMID)
	}

	errOut := cmd.ErrOrStderr()
	fmt.Fprintln(errOut, "==> Provision: starting")

	client, err := provisionClientFactory(cfg)
	if err != nil {
		return newExitError(3, err.Error())
	}

	res, err := client.ProvisionVM(context.Background(), proxmox.ProvisionRequest{
		Config:     cfg,
		RemoteDisk: remoteDisk,
	})
	if err != nil {
		msg := err.Error()
		if strings.HasPrefix(msg, "vm provision failed and cleanup failed:") {
			return newExitError(4, msg)
		}
		if !strings.HasPrefix(msg, "vm provision failed:") {
			msg = "vm provision failed: " + msg
		}
		return newExitError(3, msg)
	}

	fmt.Fprintf(errOut, "==> Provision: done (vmid=%d name=%s)\n", res.VMID, res.Name)
	return nil
}
