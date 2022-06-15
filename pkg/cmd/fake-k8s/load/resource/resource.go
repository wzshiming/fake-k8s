package resource

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/log"
	"github.com/wzshiming/fake-k8s/pkg/resource/load"
	"github.com/wzshiming/fake-k8s/pkg/runtime"
	"github.com/wzshiming/fake-k8s/pkg/utils"
	"github.com/wzshiming/fake-k8s/pkg/vars"
)

type flagpole struct {
	Name string
	File string
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(logger log.Logger) *cobra.Command {
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
	cmd.Flags().StringVar(&flags.Name, "name", "default", "cluster name")
	cmd.Flags().StringVarP(&flags.File, "file", "f", "", "resource file")
	return cmd
}

func runE(ctx context.Context, logger log.Logger, flags *flagpole) error {
	controllerName := "kube-controller-manager"
	name := vars.ProjectName + "-" + flags.Name
	workdir := utils.PathJoin(vars.TempDir, flags.Name)

	dc, err := runtime.Load(name, workdir, logger)
	if err != nil {
		return err
	}

	logger.Printf("Stopping controller %q on %q", controllerName, name)
	err = dc.Stop(ctx, controllerName)
	if err != nil {
		return err
	}
	defer func() {
		logger.Printf("Starting controller %q on %q", controllerName, name)
		err = dc.Start(ctx, controllerName)
		if err != nil {
			logger.Printf("Error starting controller %q on %q: %v", controllerName, name, err)
		}
	}()
	kubeconfig, err := dc.InHostKubeconfig()
	if err != nil {
		return err
	}

	file := flags.File
	if file == "-" {
		file = "STDIN"
	}
	logger.Printf("Loading resource %q on %q", file, name)
	err = load.Load(ctx, dc, kubeconfig, flags.File)
	if err != nil {
		return err
	}

	return nil
}
