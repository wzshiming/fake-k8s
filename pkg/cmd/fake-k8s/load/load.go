package load

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/load/resource"
	"github.com/wzshiming/fake-k8s/pkg/log"
)

// NewCommand returns a new cobra.Command for load
func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "load",
		Short: "Loads one of [resource]",
		Long:  "Loads one of [resource]",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("subcommand is required")
		},
	}
	cmd.AddCommand(resource.NewCommand(logger))
	return cmd
}
