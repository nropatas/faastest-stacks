package utils

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

func ExecCmd(env []string, dir string, bin string, command ...string) (string, string, error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	cwd := dir

	cmd := exec.Command(bin, command...)
	cmd.Dir = cwd
	cmd.Env = append(cmd.Env, os.Environ()...)
	for _, envVar := range env {
		cmd.Env = append(cmd.Env, envVar)
	}

	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		return "", "", err
	}

	os.Stdout.WriteString(stdoutBuf.String())
	os.Stderr.WriteString(stderrBuf.String())

	return strings.TrimSpace(stdoutBuf.String()), strings.TrimSpace(stderrBuf.String()), err
}

func IsAllTrue(s []bool) bool {
	if len(s) < 1 {
		return false
	}

	out := true
	for _, i := range s {
		out = out && i
	}

	return out
}
