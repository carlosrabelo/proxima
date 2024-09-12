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

var uploadFactory = defaultUploadFactory

func defaultUploadFactory(cfg config.Config) (*upload.Uploader, error) {
	client, err := proxmox.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &upload.Uploader{Client: client}, nil
}

// SetUploadFactoryForTest replaces the uploader factory (tests only).
func SetUploadFactoryForTest(fn func(config.Config) (*upload.Uploader, error)) {
	uploadFactory = fn
}

// ResetUploadFactoryForTest restores the default uploader factory.
func ResetUploadFactoryForTest() {
	uploadFactory = defaultUploadFactory
}

func newUploadCommand() *cobra.Command {
	f := &configFlags{}
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload a local qcow2 to the Proxmox node",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpload(cmd, f)
		},
	}
	bindConfigFlags(cmd, f)
	return cmd
}

func runUpload(cmd *cobra.Command, f *configFlags) error {
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

	errOut := cmd.ErrOrStderr()
	fmt.Fprintln(errOut, "==> Upload: starting")

	u, err := uploadFactory(cfg)
	if err != nil {
		return newExitError(3, err.Error())
	}

	res, err := u.UploadQcow2(context.Background(), cfg.OutputQcow2, cfg.VMID)
	if err != nil {
		msg := err.Error()
		if !strings.HasPrefix(msg, "disk upload failed:") {
			msg = "disk upload failed: " + msg
		}
		return newExitError(3, msg)
	}

	fmt.Fprintf(errOut, "==> Upload: done (%s, %d bytes)\n", res.RemotePath, res.Bytes)
	return nil
}
