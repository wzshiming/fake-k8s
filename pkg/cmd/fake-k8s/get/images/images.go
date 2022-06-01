package images

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/runtime/list"
)

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(logger logr.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "images",
		Short: "Lists images used by fake cluster",
		Long:  "Lists images used by fake cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(logger)
		},
	}
	return cmd
}

func runE(logger logr.Logger) error {
	images, err := list.ListImages()
	if err != nil {
		return err
	}
	for _, image := range images {
		fmt.Println(image)
	}
	return nil
}
