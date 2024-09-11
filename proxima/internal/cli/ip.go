package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"proxima/proxima/internal/config"
	"proxima/proxima/internal/proxmox"
)

var ipClientFactory = defaultIPClientFactory

func defaultIPClientFactory(cfg config.Config) (*proxmox.Client, error) {
	return proxmox.NewClient(cfg)
}

// SetIPClientFactoryForTest replaces the Proxmox client factory (tests only).
func SetIPClientFactoryForTest(fn func(config.Config) (*proxmox.Client, error)) {
	ipClientFactory = fn
}

// ResetIPClientFactoryForTest restores the default client factory.
func ResetIPClientFactoryForTest() {
	ipClientFactory = defaultIPClientFactory
}

func newIPCommand() *cobra.Command {
	f := &configFlags{}
	cmd := &cobra.Command{
		Use:   "ip",
		Short: "Wait for and print the guest IPv4 address",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIP(cmd, f)
		},
	}
	bindConfigFlags(cmd, f)
	return cmd
}

func runIP(cmd *cobra.Command, f *configFlags) error {
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

	errOut := cmd.ErrOrStderr()
	fmt.Fprintln(errOut, "==> WaitIP: starting")

	client, err := ipClientFactory(cfg)
	if err != nil {
		return newExitError(3, err.Error())
	}

	ip, err := client.WaitGuestIPv4(context.Background(), cfg.VMID, cfg.AgentWaitTimeout, cfg.AgentPollInterval)
	if err != nil {
		return newExitError(3, err.Error())
	}
	fmt.Fprintf(errOut, "==> WaitIP: done (%s)\n", ip)
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "ip=%s\n", ip)
	return err
}
