package logs

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
	Name   string
	Follow bool
}

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(logger log.Logger) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.ExactArgs(1),
		Use:   "logs",
		Short: "Logs one of [etcd, kube-apiserver, kube-controller-manager, kube-scheduler, fake-kubelet, prometheus]",
		Long:  "Logs one of [etcd, kube-apiserver, kube-controller-manager, kube-scheduler, fake-kubelet, prometheus]",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = vars.DefaultCluster
			return runE(cmd.Context(), logger, flags, args)
		},
	}
	cmd.Flags().BoolVarP(&flags.Follow, "follow", "f", false, "follow the log")
	return cmd
}

func runE(ctx context.Context, logger log.Logger, flags *flagpole, args []string) error {
	name := vars.ProjectName + "-" + flags.Name
	workdir := utils.PathJoin(vars.TempDir, flags.Name)

	dc, err := runtime.Load(name, workdir, logger)
	if err != nil {
		return err
	}

	if flags.Follow {
		err = dc.LogsFollow(ctx, args[0], os.Stdout)
	} else {
		err = dc.Logs(ctx, args[0], os.Stdout)
	}
	if err != nil {
		return err
	}
	return nil
}
