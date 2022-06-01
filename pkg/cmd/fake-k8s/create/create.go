package create

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/create/cluster"
)

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(logger logr.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "create",
		Short: "Creates one of [cluster]",
		Long:  "Creates one of fake Kubernetes cluster (cluster)",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := cmd.Help()
			if err != nil {
				return err
			}
			return fmt.Errorf("subcommand is required")
		},
	}
	cmd.AddCommand(cluster.NewCommand(logger))
	return cmd
}
