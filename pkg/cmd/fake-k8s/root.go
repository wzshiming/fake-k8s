package fakek8s

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/create"
	del "github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/delete"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/get"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/kubectl"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/load"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/logs"
	"github.com/wzshiming/fake-k8s/pkg/log"
	"github.com/wzshiming/fake-k8s/pkg/vars"
)

// NewCommand returns a new cobra.Command for root
func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "fake-k8s [command]",
		Short: "fake-k8s is a fake k8s",
		Long:  `fake-k8s is a fake k8s`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("subcommand is required")
		},
	}

	cmd.PersistentFlags().StringVar(&vars.DefaultCluster, "name", "default", "cluster name")
	cmd.TraverseChildren = true

	cmd.AddCommand(
		create.NewCommand(logger),
		del.NewCommand(logger),
		get.NewCommand(logger),
		load.NewCommand(logger),
		kubectl.NewCommand(logger),
		logs.NewCommand(logger),
	)
	return cmd
}
