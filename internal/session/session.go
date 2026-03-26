package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID          string
	SandboxRoot string // canonical (EvalSymlinks) absolute path
	CWD         string // canonical absolute path within SandboxRoot
	TaskID      string
	TaskJSON    string
	IssuedAt    time.Time
	CreatedAt   time.Time
}

func New(sandboxBase string) (*Session, error) {
	id := uuid.New().String()
	root := filepath.Join(sandboxBase, id)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create sandbox: %w", err)
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve sandbox root: %w", err)
	}
	return &Session{
		ID:          id,
		SandboxRoot: canonical,
		CWD:         canonical,
		CreatedAt:   time.Now(),
	}, nil
}

func (s *Session) Destroy() error {
	return os.RemoveAll(s.SandboxRoot)
}

func (s *Session) ChangeCWD(target string) error {
	resolved, err := ResolvePath(s.SandboxRoot, s.CWD, target)
	if err != nil {
		return err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return fmt.Errorf("cd: %s: No such file or directory", target)
	}
	if !info.IsDir() {
		return fmt.Errorf("cd: %s: Not a directory", target)
	}
	s.CWD = resolved
	return nil
}

// ContainPath checks that target (already resolved to absolute) is within sandboxRoot.
// Both paths must be canonical (EvalSymlinks already applied).
func ContainPath(sandboxRoot, target string) error {
	if !strings.HasPrefix(target, sandboxRoot+string(filepath.Separator)) && target != sandboxRoot {
		return fmt.Errorf("path %q escapes sandbox %q", target, sandboxRoot)
	}
	return nil
}

// ResolvePath resolves target relative to cwd with canonical containment.
// For existing paths: EvalSymlinks the full path.
// For non-existing paths: EvalSymlinks the nearest existing ancestor, then check remainder has no "..".
func ResolvePath(sandboxRoot, cwd, target string) (string, error) {
	var abs string
	if filepath.IsAbs(target) {
		abs = filepath.Clean(target)
	} else {
		abs = filepath.Clean(filepath.Join(cwd, target))
	}

	// Try EvalSymlinks on the full path (works if it exists)
	canonical, err := filepath.EvalSymlinks(abs)
	if err == nil {
		if err := ContainPath(sandboxRoot, canonical); err != nil {
			return "", err
		}
		return canonical, nil
	}

	// Path doesn't fully exist — walk up to find nearest existing ancestor
	remaining := []string{}
	dir := abs
	for {
		canonical, err := filepath.EvalSymlinks(dir)
		if err == nil {
			// Found existing ancestor. Check containment.
			if err := ContainPath(sandboxRoot, canonical); err != nil {
				return "", err
			}
			// Check remaining segments don't contain ".."
			for _, seg := range remaining {
				if seg == ".." {
					return "", fmt.Errorf("path %q contains .. after resolved ancestor", target)
				}
			}
			// Reconstruct: canonical ancestor + remaining segments
			parts := append([]string{canonical}, remaining...)
			return filepath.Join(parts...), nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("cannot resolve any ancestor of %q", target)
		}
		remaining = append([]string{filepath.Base(dir)}, remaining...)
		dir = parent
	}
}

// RelativeCWD returns CWD relative to SandboxRoot for display.
func (s *Session) RelativeCWD() string {
	rel, err := filepath.Rel(s.SandboxRoot, s.CWD)
	if err != nil {
		return s.CWD
	}
	if rel == "." {
		return ""
	}
	return rel
}
