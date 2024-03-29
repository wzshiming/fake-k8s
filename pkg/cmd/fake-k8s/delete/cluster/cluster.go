package cluster

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/log"
	"github.com/wzshiming/fake-k8s/pkg/runtime"
	"github.com/wzshiming/fake-k8s/pkg/utils"
	"github.com/wzshiming/fake-k8s/pkg/vars"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(logger log.Logger) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Deletes a cluster",
		Long:  "Deletes a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = vars.DefaultCluster
			return runE(cmd.Context(), logger, flags)
		},
	}
	return cmd
}

func runE(ctx context.Context, logger log.Logger, flags *flagpole) error {
	name := vars.ProjectName + "-" + flags.Name
	workdir := utils.PathJoin(vars.TempDir, flags.Name)

	dc, err := runtime.Load(name, workdir, logger)
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
