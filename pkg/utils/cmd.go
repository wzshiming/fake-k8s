package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func ForkExec(ctx context.Context, dir string, name string, arg ...string) error {
	pidPath := filepath.Join(dir, "pids", filepath.Base(name)+".pid")
	pidData, err := os.ReadFile(pidPath)
	if err == nil {
		_, err = strconv.Atoi(string(pidData))
		if err == nil {
			return nil // already running
		}
	}

	logPath := filepath.Join(dir, "logs", filepath.Base(name)+".log")
	cmdlinesPath := filepath.Join(dir, "cmdlines", filepath.Base(name))

	logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", logPath, err)
	}

	args := append([]string{name}, arg...)

	err = os.WriteFile(cmdlinesPath, []byte(strings.Join(args, " ")), 0644)
	if err != nil {
		return fmt.Errorf("write cmdline file %s: %w", cmdlinesPath, err)
	}

	cmd := commandStart(ctx, args[0], args[1:]...)
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
	cmdlinesPath := filepath.Join(dir, "cmdlines", filepath.Base(name))

	data, err := os.ReadFile(cmdlinesPath)
	if err != nil {
		return err
	}

	args := strings.Split(string(data), " ")

	return ForkExec(ctx, dir, args[0], args...)
}

func ForkExecKill(ctx context.Context, dir string, name string) error {
	pidPath := filepath.Join(dir, "pids", filepath.Base(name)+".pid")
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
