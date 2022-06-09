package cluster

import (
	"context"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/runtime"
	"github.com/wzshiming/fake-k8s/pkg/vars"
)

type flagpole struct {
	Name string
	Wait time.Duration
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(logger logr.Logger) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Deletes a cluster",
		Long:  "Deletes a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), logger, flags)
		},
	}
	cmd.Flags().StringVar(&flags.Name, "name", "default", "cluster name, config")
	cmd.Flags().DurationVar(&flags.Wait, "wait", time.Duration(0), "wait for control plane node to be ready")
	return cmd
}

func runE(ctx context.Context, logger logr.Logger, flags *flagpole) error {
	name := vars.ProjectName + "-" + flags.Name
	workdir := filepath.Join(vars.TempDir, flags.Name)

	dc, err := runtime.Load(name, workdir)
	if err != nil {
		return err
	}
	logger.Info("stop cluster", "cluster", name)
	err = dc.Down(ctx)
	if err != nil {
		logger.Info("Failed stop cluster", "cluster", name, "err", err)
	}

	logger.Info("delete cluster", "cluster", name)
	err = dc.Uninstall(ctx)
	if err != nil {
		return err
	}
	logger.Info("cluster cleaned", "cluster", name)
	return nil
}
