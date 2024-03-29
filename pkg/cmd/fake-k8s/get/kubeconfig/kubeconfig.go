package kubeconfig

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/log"
	"github.com/wzshiming/fake-k8s/pkg/runtime"
	"github.com/wzshiming/fake-k8s/pkg/utils"
	"github.com/wzshiming/fake-k8s/pkg/vars"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(logger log.Logger) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "kubeconfig",
		Short: "Prints cluster kubeconfig",
		Long:  "Prints cluster kubeconfig",
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

	err = dc.KubectlInCluster(ctx, utils.IOStreams{
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}, "config", "view", "--minify", "--flatten", "--raw")

	if err != nil {
		return err
	}
	return nil
}
