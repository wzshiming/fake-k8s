package kubectl

import (
	"context"
	"os/exec"

	"github.com/wzshiming/fake-k8s/pkg/utils"
)

// Run runs a command in the given context.
func Run(ctx context.Context, iostm utils.IOStreams, args ...string) error {
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdout = iostm.Out
	cmd.Stderr = iostm.ErrOut
	cmd.Stdin = iostm.In
	return cmd.Run()
}
