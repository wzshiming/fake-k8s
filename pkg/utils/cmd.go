package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func ForkExec(ctx context.Context, dir string, name string, arg ...string) error {
	pidPath := PathJoin(dir, "pids", filepath.Base(name)+".pid")
	pidData, err := os.ReadFile(pidPath)
	if err == nil {
		_, err = strconv.Atoi(string(pidData))
		if err == nil {
			return nil // already running
		}
	}

	logPath := PathJoin(dir, "logs", filepath.Base(name)+".log")
	cmdlinePath := PathJoin(dir, "cmdline", filepath.Base(name))

	err = os.MkdirAll(filepath.Dir(pidPath), 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(logPath), 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(cmdlinePath), 0755)
	if err != nil {
		return err
	}

	logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", logPath, err)
	}

	args := append([]string{name}, arg...)

	err = os.WriteFile(cmdlinePath, []byte(strings.Join(args, "\x00")), 0644)
	if err != nil {
		return fmt.Errorf("write cmdline file %s: %w", cmdlinePath, err)
	}

	cmd := startProcess(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	err = cmd.Start()
	if err != nil {
		return err
	}

	err = os.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)
	if err != nil {
		return fmt.Errorf("write pid file %s: %w", pidPath, err)
	}
	return nil
}

func ForkExecRestart(ctx context.Context, dir string, name string) error {
	cmdlinePath := PathJoin(dir, "cmdline", filepath.Base(name))

	data, err := os.ReadFile(cmdlinePath)
	if err != nil {
		return err
	}

	args := strings.Split(string(data), "\x00")

	return ForkExec(ctx, dir, args[0], args...)
}

func ForkExecKill(ctx context.Context, dir string, name string) error {
	pidPath := PathJoin(dir, "pids", filepath.Base(name)+".pid")
	if _, err := os.Stat(pidPath); err != nil {
		return nil
	}
	raw, err := os.ReadFile(pidPath)
	if err != nil {
		return fmt.Errorf("read pid file %s: %w", pidPath, err)
	}
	pid, err := strconv.Atoi(string(raw))
	if err != nil {
		return fmt.Errorf("parse pid file %s: %w", pidPath, err)
	}
	err = killProcess(ctx, pid)
	if err != nil {
		return err
	}
	err = os.Remove(pidPath)
	if err != nil {
		return err
	}
	return nil
}

func Exec(ctx context.Context, dir string, stm IOStreams, name string, arg ...string) error {
	cmd := command(ctx, name, arg...)
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

type IOStreams struct {
	// In think, os.Stdin
	In io.Reader
	// Out think, os.Stdout
	Out io.Writer
	// ErrOut think, os.Stderr
	ErrOut io.Writer
}

func killProcess(ctx context.Context, pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	err = process.Kill()
	if err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		return fmt.Errorf("kill process: %w", err)
	}
	process.Wait()
	return nil
}
