package clusters

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/runtime/list"
	"github.com/wzshiming/fake-k8s/pkg/vars"
)

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(logger logr.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "clusters",
		Short: "Lists existing fake clusters by their name",
		Long:  "Lists existing fake clusters by their name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(logger)
		},
	}
	return cmd
}

func runE(logger logr.Logger) error {
	clusters, err := list.ListClusters(vars.TempDir)
	if err != nil {
		return err
	}
	for _, cluster := range clusters {
		fmt.Println(cluster)
	}
	return nil
}
