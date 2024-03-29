package delete

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/delete/cluster"
	"github.com/wzshiming/fake-k8s/pkg/log"
)

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "delete",
		Short: "Deletes one of [cluster]",
		Long:  "Deletes one of [cluster]",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("subcommand is required")
		},
	}
	cmd.AddCommand(cluster.NewCommand(logger))
	return cmd
}
