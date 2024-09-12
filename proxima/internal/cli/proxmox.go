package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"proxima/proxima/internal/config"
	"proxima/proxima/internal/proxmox"
)

func newProxmoxCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxmox",
		Short: "Proxmox API helpers",
	}
	cmd.AddCommand(newProxmoxPingCommand())
	return cmd
}

func newProxmoxPingCommand() *cobra.Command {
	f := &configFlags{}
	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Authenticate and print Proxmox API version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProxmoxPing(cmd, f)
		},
	}
	bindConfigFlags(cmd, f)
	return cmd
}

func runProxmoxPing(cmd *cobra.Command, f *configFlags) error {
	path, err := resolveConfigPath(f.configPath)
	if err != nil {
		return err
	}

	cfg, err := config.Load(path, overrideFromFlags(cmd, f))
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	client, err := proxmox.NewClient(cfg)
	if err != nil {
		return newExitError(3, err.Error())
	}

	ctx := context.Background()
	version, err := client.Version(ctx)
	if err != nil {
		return newExitError(3, err.Error())
	}
	if _, err := client.NodeStatus(ctx); err != nil {
		return newExitError(3, err.Error())
	}

	_, err = fmt.Fprintf(cmd.OutOrStdout(), "proxmox: ok\nversion: %s\n", version)
	return err
}
