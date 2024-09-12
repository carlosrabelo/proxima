package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"proxima/proxima/internal/config"
	"proxima/proxima/internal/proxmox"
)

var startClientFactory = defaultStartClientFactory

func defaultStartClientFactory(cfg config.Config) (*proxmox.Client, error) {
	return proxmox.NewClient(cfg)
}

// SetStartClientFactoryForTest replaces the Proxmox client factory (tests only).
func SetStartClientFactoryForTest(fn func(config.Config) (*proxmox.Client, error)) {
	startClientFactory = fn
}

// ResetStartClientFactoryForTest restores the default client factory.
func ResetStartClientFactoryForTest() {
	startClientFactory = defaultStartClientFactory
}

func newStartCommand() *cobra.Command {
	f := &configFlags{}
	waitIP := true
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a Proxmox VM and optionally wait for guest IPv4",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(cmd, f, waitIP)
		},
	}
	bindConfigFlags(cmd, f)
	cmd.Flags().BoolVar(&waitIP, "wait-ip", true, "wait for guest IPv4 after start")
	return cmd
}

func runStart(cmd *cobra.Command, f *configFlags, waitIP bool) error {
	path, err := resolveConfigPath(f.configPath)
	if err != nil {
		return err
	}

	o := overrideFromFlags(cmd, f)
	if cmd.Flags().Changed("wait-ip") {
		o.WaitIP = &waitIP
	}
	cfg, err := config.Load(path, o)
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}
	if cmd.Flags().Changed("wait-ip") {
		cfg.WaitIP = waitIP
	}

	errOut := cmd.ErrOrStderr()
	fmt.Fprintln(errOut, "==> Start: starting")

	client, err := startClientFactory(cfg)
	if err != nil {
		return newExitError(3, err.Error())
	}

	if err := client.StartVM(context.Background(), cfg.VMID); err != nil {
		msg := err.Error()
		if !strings.HasPrefix(msg, "vm start failed:") {
			msg = "vm start failed: " + msg
		}
		return newExitError(3, msg)
	}
	fmt.Fprintf(errOut, "==> Start: done (vmid=%d)\n", cfg.VMID)

	if !cfg.WaitIP {
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "vmid=%d started\n", cfg.VMID)
		return err
	}

	fmt.Fprintln(errOut, "==> WaitIP: starting")
	ip, err := client.WaitGuestIPv4(context.Background(), cfg.VMID, cfg.AgentWaitTimeout, cfg.AgentPollInterval)
	if err != nil {
		return newExitError(3, err.Error())
	}
	fmt.Fprintf(errOut, "==> WaitIP: done (%s)\n", ip)
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "vmid=%d ip=%s\n", cfg.VMID, ip)
	return err
}
