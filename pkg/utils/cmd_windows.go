//go:build windows

package utils

import (
	"context"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func startProcess(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := command(ctx, name, arg...)
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NEW_CONSOLE
	return cmd
}

func command(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	return cmd
}
