package cluster

import (
	"context"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/runtime/compose"
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
	dc := compose.NewCluster(vars.ProjectName+"-"+flags.Name, filepath.Join(vars.TempDir, flags.Name), vars.Runtime)
	err := dc.Down(ctx)
	if err != nil {
		return err
	}
	err = dc.Uninstall(ctx)
	if err != nil {
		return err
	}
	return nil
}
