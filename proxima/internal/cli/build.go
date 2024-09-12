package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"proxima/proxima/internal/config"
	"proxima/proxima/internal/diskbuild"
)

// buildDockerFactory is a test seam for injecting a fake DockerRunner.
var buildDockerFactory = defaultBuildDockerFactory

func defaultBuildDockerFactory() diskbuild.DockerRunner {
	return diskbuild.ExecDocker{}
}

// SetBuildDockerFactoryForTest replaces the Docker runner factory (tests only).
func SetBuildDockerFactoryForTest(fn func() diskbuild.DockerRunner) {
	buildDockerFactory = fn
}

// ResetBuildDockerFactoryForTest restores the default Docker runner factory.
func ResetBuildDockerFactoryForTest() {
	buildDockerFactory = defaultBuildDockerFactory
}

func newBuildCommand() *cobra.Command {
	f := &configFlags{}
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a local Alpine Docker-ready qcow2 image",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild(cmd, f)
		},
	}
	bindConfigFlags(cmd, f)
	return cmd
}

func runBuild(cmd *cobra.Command, f *configFlags) error {
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
	fmt.Fprintln(errOut, "==> BuildLocal: starting")

	b := &diskbuild.Builder{Docker: buildDockerFactory()}
	res, err := b.Build(context.Background(), cfg)
	if err != nil {
		msg := err.Error()
		if !strings.HasPrefix(msg, "local disk build failed:") {
			msg = "local disk build failed: " + msg
		}
		return newExitError(2, msg)
	}

	fmt.Fprintf(errOut, "==> BuildLocal: done (%s, %d bytes)\n", res.Qcow2Path, res.Bytes)
	return nil
}
