package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func Exec(ctx context.Context, dir string, stm IOStreams, name string, arg ...string) error {
	cmd := exec.CommandContext(ctx, name, arg...)
	cmd.Dir = dir
	cmd.Stdin = stm.In
	cmd.Stdout = stm.Out
	cmd.Stderr = stm.ErrOut

	if cmd.Stderr == nil {
		buf := bytes.NewBuffer(nil)
		cmd.Stderr = buf
	}
	err := cmd.Run()
	if err != nil {
		if buf, ok := cmd.Stderr.(*bytes.Buffer); ok {
			return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(arg, " "), err, buf.String())
		}
		return fmt.Errorf("%s %s: %w", name, strings.Join(arg, " "), err)
	}
	return nil
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
