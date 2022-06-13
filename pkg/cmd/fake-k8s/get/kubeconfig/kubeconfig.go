package kubeconfig

import (
	"context"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/cmd"
	"github.com/wzshiming/fake-k8s/pkg/runtime"
	"github.com/wzshiming/fake-k8s/pkg/vars"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(logger cmd.Logger) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "kubeconfig",
		Short: "Prints cluster kubeconfig",
		Long:  "Prints cluster kubeconfig",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), logger, flags)
		},
	}
	cmd.Flags().StringVar(&flags.Name, "name", "default", "cluster name")
	return cmd
}

func runE(ctx context.Context, logger cmd.Logger, flags *flagpole) error {
	kubeconfigPath := filepath.Join(vars.TempDir, flags.Name, runtime.InHostKubeconfigName)

	data, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return err
	}
	os.Stdout.Write(data)
	return nil
}
