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
	All  bool
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(logger log.Logger) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Use:   "cluster [name ...]",
		Short: "Deletes a cluster",
		Long:  "Deletes a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.All {
				list, err := runtime.ListClusters(vars.TempDir)
				if err != nil {
					return err
				}
				for _, name := range list {
					flags.Name = name
					err = runE(cmd.Context(), logger, flags)
					if err != nil {
						logger.Printf("Error deleting cluster %q: %v", name, err)
					}
				}
				return nil
			}
			if len(args) == 0 {
				flags.Name = vars.DefaultCluster
			} else if len(args) == 1 {
				flags.Name = args[0]
			} else {
				for _, name := range args {
					flags.Name = name
					err := runE(cmd.Context(), logger, flags)
					if err != nil {
						logger.Printf("Error deleting cluster %q: %v", name, err)
					}
				}
				return nil
			}
			return runE(cmd.Context(), logger, flags)
		},
	}
	cmd.Flags().BoolVar(&flags.All, "all", false, "Delete all clusters")
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
