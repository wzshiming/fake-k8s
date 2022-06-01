package utils

import (
	"context"
	"io"
	"os"
	"os/exec"
)

func Exec(ctx context.Context, dir string, name string, arg ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, arg...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

func LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

type IOStreams struct {
	// In think, os.Stdin
	In io.Reader
	// Out think, os.Stdout
	Out io.Writer
	// ErrOut think, os.Stderr
	ErrOut io.Writer
}
