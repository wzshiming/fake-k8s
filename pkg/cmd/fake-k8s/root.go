package fakek8s

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/create"
	del "github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/delete"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/get"
	"github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s/load"
	"github.com/wzshiming/fake-k8s/pkg/log"
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

	cmd.AddCommand(
		create.NewCommand(logger),
		del.NewCommand(logger),
		get.NewCommand(logger),
		load.NewCommand(logger),
	)
	return cmd
}
