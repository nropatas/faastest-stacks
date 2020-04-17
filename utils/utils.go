package utils

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
)

func ExecCmd(env []string, dir string, bin string, command ...string) (string, string, error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	var errStdout, errStderr error

	cwd := dir

	cmd := exec.Command(bin, command...)
	cmd.Dir = cwd
	cmd.Env = append(cmd.Env, os.Environ()...)
	for _, envVar := range env {
		cmd.Env = append(cmd.Env, envVar)
	}

	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)
	err := cmd.Start()
	if err != nil {
		return "", "", err
	}

	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
	}()

	go func() {
		_, errStderr = io.Copy(stderr, stderrIn)
	}()

	err = cmd.Wait()
	if errStdout != nil || errStderr != nil {
		return "", "", errors.New("failed to capture stdout or stderr")
	}

	return strings.TrimSpace(stdoutBuf.String()), strings.TrimSpace(stderrBuf.String()), err
}
