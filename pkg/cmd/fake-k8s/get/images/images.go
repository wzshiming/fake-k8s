package images

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/log"
	"github.com/wzshiming/fake-k8s/pkg/runtime"
	"github.com/wzshiming/fake-k8s/pkg/vars"
)

type flagpole struct {
	Runtime string
}

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(logger log.Logger) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "images",
		Short: "Lists images used by fake cluster, only for docker/nerdctl/kind runtime",
		Long:  "Lists images used by fake cluster, only for docker/nerdctl/kind runtime",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(logger, flags)
		},
	}
	cmd.Flags().StringVar(&flags.Runtime, "runtime", vars.Runtime, "runtime of the fake cluster ("+strings.Join(runtime.List(), " or ")+")")
	return cmd
}

func runE(logger log.Logger, flags *flagpole) error {
	var images []string
	var err error
	switch flags.Runtime {
	case "docker", "nerdctl":
		images, err = runtime.ListImagesCompose()
	case "kind":
		images, err = runtime.ListImagesKind()
	case "binary":
		logger.Printf("no images need to be pull for %s", flags.Runtime)
		return nil
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
