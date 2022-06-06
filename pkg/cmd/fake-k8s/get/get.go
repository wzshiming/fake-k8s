package get

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/get/clusters"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/get/images"
)

// NewCommand returns a new cobra.Command for get
func NewCommand(logger logr.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "get",
		Short: "Gets one of [clusters, images]",
		Long:  "Gets one of [clusters, images]",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := cmd.Help()
			if err != nil {
				return err
			}
			return fmt.Errorf("subcommand is required")
		},
	}
	// add subcommands
	cmd.AddCommand(clusters.NewCommand(logger))
	cmd.AddCommand(images.NewCommand(logger))
	return cmd
}