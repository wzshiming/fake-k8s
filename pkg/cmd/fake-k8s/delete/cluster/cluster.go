package cluster

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/cmd"
	"github.com/wzshiming/fake-k8s/pkg/runtime"
	"github.com/wzshiming/fake-k8s/pkg/utils"
	"github.com/wzshiming/fake-k8s/pkg/vars"
)

type flagpole struct {
	Name string
	Wait time.Duration
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(logger cmd.Logger) *cobra.Command {
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
	cmd.Flags().StringVar(&flags.Name, "name", "default", "cluster name")
	cmd.Flags().DurationVar(&flags.Wait, "wait", time.Duration(0), "wait for control plane node to be ready")
	return cmd
}

func runE(ctx context.Context, logger cmd.Logger, flags *flagpole) error {
	name := vars.ProjectName + "-" + flags.Name
	workdir := utils.PathJoin(vars.TempDir, flags.Name)

	dc, err := runtime.Load(name, workdir)
	if err != nil {
		return err
	}
	logger.Printf("Stopping cluster %q", name)
	err = dc.Down(ctx)
	if err != nil {
		logger.Printf("Error stopping cluster %q: %v", name, err)
	}

	logger.Printf("Deleting cluster %q", name)
	err = dc.Uninstall(ctx)
	if err != nil {
		return err
	}
	logger.Printf("Cluster %q deleted", name)
	return nil
}
