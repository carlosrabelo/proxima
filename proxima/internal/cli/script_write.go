package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"proxima/proxima/internal/script"
)

func newScriptCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "script",
		Short: "Alpine provision script helpers",
	}
	cmd.AddCommand(newScriptWriteCommand())
	return cmd
}

func newScriptWriteCommand() *cobra.Command {
	var rootPassword string
	var guestSSHPublicKey string
	var output string

	cmd := &cobra.Command{
		Use:   "write",
		Short: "Write generated Alpine setup.sh",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			password := rootPassword
			if password == "" {
				password = os.Getenv("PROXIMA_ROOT_PASSWORD")
			}
			params := script.Params{RootPassword: password}
			if guestSSHPublicKey != "" {
				key, err := script.ReadPublicKeyFile(guestSSHPublicKey)
				if err != nil {
					return err
				}
				params.SSHAuthorizedKey = key
			}
			if err := script.WriteFile(output, params); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", output)
			return err
		},
	}

	cmd.Flags().StringVar(&rootPassword, "root-password", "", "guest root password")
	cmd.Flags().StringVar(&guestSSHPublicKey, "guest-ssh-public-key", "", "path to OpenSSH public key for guest root")
	cmd.Flags().StringVarP(&output, "output", "o", "setup.sh", "output path for setup.sh")
	return cmd
}
