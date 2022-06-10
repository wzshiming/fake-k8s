package binaries

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/runtime"
	"github.com/wzshiming/fake-k8s/pkg/vars"
)

type flagpole struct {
	Runtime string
}

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(logger logr.Logger) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "binaries",
		Short: "Lists binaries used by fake cluster, only for binary runtime",
		Long:  "Lists binaries used by fake cluster, only for binary runtime",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(logger, flags)
		},
	}
	cmd.Flags().StringVar(&flags.Runtime, "runtime", vars.Runtime, "runtime of the fake cluster ("+strings.Join(runtime.List(), " or ")+")")
	return cmd
}

func runE(logger logr.Logger, flags *flagpole) error {
	var images []string
	var err error
	switch flags.Runtime {
	case "docker", "nertctl", "kind":
		images = nil
	case "binary":
		images, err = runtime.ListBinaries()
	default:
		return fmt.Errorf("unknown runtime: %s", flags.Runtime)
	}

	if err != nil {
		return err
	}
	for _, image := range images {
		fmt.Println(image)
	}
	return nil
}
