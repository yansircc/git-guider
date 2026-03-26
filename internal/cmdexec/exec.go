package cmdexec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"git-guider/internal/session"
)

type Executor struct {
	SandboxRoot string
	CWD         string
	Env         []string
}

func NewExecutor(sandboxRoot, cwd string) *Executor {
	EnsureGitConfig(sandboxRoot)
	return &Executor{
		SandboxRoot: sandboxRoot,
		CWD:         cwd,
		Env:         BaselineEnv(sandboxRoot),
	}
}

type ExecResult struct {
	Stdout string
	Stderr string
	CWD    string // possibly changed by cd
}

func (e *Executor) Run(cmdLine string) (*ExecResult, error) {
	cmd, err := Parse(cmdLine)
	if err != nil {
		return nil, err
	}

	if err := ValidateCommand(cmd, e.SandboxRoot, e.CWD); err != nil {
		return nil, fmt.Errorf("denied: %w", err)
	}

	switch cmd.Program {
	case "cd":
		return e.handleCD(cmd)
	case "pwd":
		return &ExecResult{Stdout: e.CWD + "\n", CWD: e.CWD}, nil
	case "clear":
		return &ExecResult{CWD: e.CWD}, nil
	case "echo":
		return e.handleEcho(cmd)
	default:
		return e.execCommand(cmd)
	}
}

func (e *Executor) handleCD(cmd *ParsedCommand) (*ExecResult, error) {
	target := e.SandboxRoot
	if len(cmd.Args) > 0 {
		target = cmd.Args[0]
	}

	var resolved string
	var err error
	if filepath.IsAbs(target) {
		resolved, err = session.ResolvePath(e.SandboxRoot, e.CWD, target)
	} else {
		resolved, err = session.ResolvePath(e.SandboxRoot, e.CWD, target)
	}
	if err != nil {
		return nil, fmt.Errorf("cd: %w", err)
	}

	info, statErr := os.Stat(resolved)
	if statErr != nil {
		return nil, fmt.Errorf("cd: %s: No such file or directory", target)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("cd: %s: Not a directory", target)
	}

	e.CWD = resolved
	return &ExecResult{CWD: e.CWD}, nil
}

func (e *Executor) handleEcho(cmd *ParsedCommand) (*ExecResult, error) {
	output := strings.Join(cmd.Args, " ") + "\n"

	if cmd.Redirect != nil {
		target, err := session.ResolvePath(e.SandboxRoot, e.CWD, cmd.Redirect.Target)
		if err != nil {
			return nil, fmt.Errorf("echo redirect: %w", err)
		}
		var f *os.File
		if cmd.Redirect.Append {
			f, err = os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		} else {
			f, err = os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		}
		if err != nil {
			return nil, fmt.Errorf("echo: %w", err)
		}
		_, werr := f.WriteString(strings.Join(cmd.Args, " ") + "\n")
		f.Close()
		if werr != nil {
			return nil, fmt.Errorf("echo: %w", werr)
		}
		return &ExecResult{CWD: e.CWD}, nil
	}

	return &ExecResult{Stdout: output, CWD: e.CWD}, nil
}

func (e *Executor) execCommand(cmd *ParsedCommand) (*ExecResult, error) {
	prog := cmd.Program
	args := cmd.Args

	// For git with -C flag, we don't need special handling since the flag
	// was already validated. The git binary will handle -C itself.

	path, err := exec.LookPath(prog)
	if err != nil {
		return nil, fmt.Errorf("%s: command not found", prog)
	}

	c := exec.Command(path, args...)
	c.Dir = e.CWD
	c.Env = e.Env

	var stdout, stderr strings.Builder
	c.Stdout = &stdout
	c.Stderr = &stderr

	runErr := c.Run()

	result := &ExecResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		CWD:    e.CWD,
	}

	if runErr != nil {
		// Return the error along with any output for user feedback
		if result.Stderr != "" {
			return result, fmt.Errorf("%s", strings.TrimSpace(result.Stderr))
		}
		return result, runErr
	}

	return result, nil
}

// RunInternal executes a command directly without algebra validation.
// Used for internal operations like getting branch name for prompt.
func (e *Executor) RunInternal(prog string, args ...string) (string, error) {
	path, err := exec.LookPath(prog)
	if err != nil {
		return "", err
	}
	c := exec.Command(path, args...)
	c.Dir = e.CWD
	c.Env = e.Env
	out, err := c.Output()
	return strings.TrimSpace(string(out)), err
}
