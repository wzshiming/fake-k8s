package get

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/cmd"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/get/binaries"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/get/clusters"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/get/images"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/get/kubeconfig"
)

// NewCommand returns a new cobra.Command for get
func NewCommand(logger cmd.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "get",
		Short: "Gets one of [clusters, images, binaries, kubeconfig]",
		Long:  "Gets one of [clusters, images, binaries, kubeconfig]",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("subcommand is required")
		},
	}
	// add subcommands
	cmd.AddCommand(clusters.NewCommand(logger))
	cmd.AddCommand(images.NewCommand(logger))
	cmd.AddCommand(binaries.NewCommand(logger))
	cmd.AddCommand(kubeconfig.NewCommand(logger))
	return cmd
}
