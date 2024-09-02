package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"proxima/proxima/internal/version"
)

// NewRootCommand builds the proxima root command tree.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "proxima",
		Short:         "Build and provision Alpine Linux Docker-ready VMs on Proxmox VE",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the proxima version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "proxima %s\n", version.Version)
			return err
		},
	})

	return root
}

// Execute runs the root command with process args and maps errors to exit codes.
func Execute() int {
	return ExecuteWithArgs(os.Args[1:], os.Stdout, os.Stderr)
}

// ExecuteWithArgs runs the root command with the given args and streams (for tests).
func ExecuteWithArgs(args []string, stdout, stderr io.Writer) int {
	root := NewRootCommand()
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
