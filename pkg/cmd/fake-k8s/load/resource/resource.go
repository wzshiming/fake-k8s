package resource

import (
	"context"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/resource/load"
	"github.com/wzshiming/fake-k8s/pkg/runtime"
	"github.com/wzshiming/fake-k8s/pkg/vars"
)

type flagpole struct {
	Name string
	File string
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(logger logr.Logger) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "resource",
		Short: "Loads a resource",
		Long:  "Loads a resource",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), logger, flags)
		},
	}
	cmd.Flags().StringVar(&flags.Name, "name", "default", "cluster name, config")
	cmd.Flags().StringVarP(&flags.File, "file", "f", "", "resource file")
	return cmd
}

func runE(ctx context.Context, logger logr.Logger, flags *flagpole) error {
	controllerName := vars.ProjectName + "-" + flags.Name + "-kube-controller-manager"
	name := vars.ProjectName + "-" + flags.Name
	workdir := filepath.Join(vars.TempDir, flags.Name)

	dc, err := runtime.Load(name, workdir)
	if err != nil {
		return err
	}

	err = dc.Stop(ctx, controllerName)
	if err != nil {
		return err
	}
	defer dc.Start(ctx, controllerName)
	kubeconfig, err := dc.InHostKubeconfig()
	if err != nil {
		return err
	}

	err = load.Load(ctx, kubeconfig, flags.File)
	if err != nil {
		return err
	}

	return nil
}
