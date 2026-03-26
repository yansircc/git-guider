package cmdexec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestExecutor(t *testing.T) (*Executor, string) {
	t.Helper()
	root := t.TempDir()
	canonical, _ := filepath.EvalSymlinks(root)
	EnsureGitConfig(canonical)
	return NewExecutor(canonical, canonical), canonical
}

// --- Path escapes ---

func TestEscapeGitDashC(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git -C / status")
	assertDenied(t, err, "git -C /")
}

func TestEscapeGitDashCRelative(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git -C ../../../ status")
	assertDenied(t, err, "git -C ../../../")
}

func TestEscapeGitDir(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git --git-dir=/tmp/x status")
	assertDenied(t, err, "git --git-dir=/tmp/x")
}

func TestEscapeGitWorkTree(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git --work-tree=/ status")
	assertDenied(t, err, "git --work-tree=/")
}

func TestEscapeGitExecPath(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git --exec-path=/tmp status")
	assertDenied(t, err, "git --exec-path")
}

func TestEscapeCd(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("cd /tmp")
	assertDenied(t, err, "cd /tmp")
}

func TestEscapeEchoRedirect(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("echo hello > /etc/passwd")
	assertDenied(t, err, "echo > /etc/passwd")
}

func TestEscapeGitCloneAbsPath(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git clone /Users/someone/repo")
	assertDenied(t, err, "git clone /Users/someone/repo")
}

// --- Transport escapes ---

func TestEscapeCloneHTTPS(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git clone https://github.com/x/y")
	assertDenied(t, err, "git clone https://")
}

func TestEscapeCloneSSH(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git clone ssh://user@host/repo")
	assertDenied(t, err, "git clone ssh://")
}

func TestEscapeCloneGitProto(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git clone git://host/repo")
	assertDenied(t, err, "git clone git://")
}

func TestEscapeCloneFileProto(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git clone file:///etc/passwd")
	assertDenied(t, err, "git clone file://")
}

func TestEscapeCloneScpStyle(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git clone user@host:repo")
	assertDenied(t, err, "git clone user@host:repo")
}

func TestEscapeRemoteAddHTTPS(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git remote add evil https://example.com/repo")
	assertDenied(t, err, "git remote add https://")
}

// --- Symlink escapes ---

func TestEscapeSymlinkCd(t *testing.T) {
	e, root := newTestExecutor(t)
	os.Symlink("/tmp", filepath.Join(root, "escape_link"))
	_, err := e.Run("cd escape_link")
	assertDenied(t, err, "cd through symlink to /tmp")
}

func TestEscapeSymlinkCat(t *testing.T) {
	e, root := newTestExecutor(t)
	os.Symlink("/etc/hosts", filepath.Join(root, "trap"))
	_, err := e.Run("cat trap")
	assertDenied(t, err, "cat through symlink")
}

func TestEscapeSymlinkEchoWrite(t *testing.T) {
	e, root := newTestExecutor(t)
	os.Symlink("/tmp", filepath.Join(root, "escape_dir"))
	_, err := e.Run("echo evil > escape_dir/file")
	assertDenied(t, err, "echo > through symlink dir")
}

// --- Config escapes ---

func TestEscapeGitCProxy(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git -c core.gitProxy=evil status")
	assertDenied(t, err, "git -c core.gitProxy")
}

func TestEscapeGitCHooksPath(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git -c core.hooksPath=/tmp status")
	assertDenied(t, err, "git -c core.hooksPath")
}

func TestEscapeGitConfigGlobal(t *testing.T) {
	e, root := newTestExecutor(t)
	// Need a git repo for config commands
	exec_git_init(e, root)
	_, err := e.Run("git config --global user.name evil")
	assertDenied(t, err, "git config --global")
}

func TestEscapeGitConfigDangerousKey(t *testing.T) {
	e, root := newTestExecutor(t)
	exec_git_init(e, root)
	_, err := e.Run("git config core.hooksPath /tmp/hooks")
	assertDenied(t, err, "git config core.hooksPath")
}

// --- Subcommand/flag escapes ---

func TestEscapeRemoteSetURL(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git remote set-url origin https://evil.com")
	assertDenied(t, err, "git remote set-url")
}

func TestEscapeCloneTemplate(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git clone --template=/tmp/hooks repo")
	assertDenied(t, err, "git clone --template")
}

func TestEscapeInitTemplate(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git init --template=/tmp/hooks")
	assertDenied(t, err, "git init --template")
}

func TestEscapeUnknownCommand(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("bash -c 'rm -rf /'")
	assertDenied(t, err, "unknown command bash")
}

// --- Positive tests (must allow) ---

func TestAllowTouchNewFile(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("touch new.txt")
	if err != nil {
		t.Errorf("touch new.txt should be allowed: %v", err)
	}
}

func TestAllowEchoNewFile(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("echo hello > new.txt")
	if err != nil {
		t.Errorf("echo > new.txt should be allowed: %v", err)
	}
}

func TestAllowGitInit(t *testing.T) {
	e, _ := newTestExecutor(t)
	_, err := e.Run("git init")
	if err != nil {
		t.Errorf("git init should be allowed: %v", err)
	}
}

func TestAllowCdSubdir(t *testing.T) {
	e, root := newTestExecutor(t)
	os.MkdirAll(filepath.Join(root, "subdir"), 0o755)
	_, err := e.Run("cd subdir")
	if err != nil {
		t.Errorf("cd subdir should be allowed: %v", err)
	}
}

// helpers

func assertDenied(t *testing.T, err error, desc string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected denial, got nil error", desc)
		return
	}
	if !strings.Contains(err.Error(), "denied") && !strings.Contains(err.Error(), "not allowed") &&
		!strings.Contains(err.Error(), "escapes sandbox") && !strings.Contains(err.Error(), "cd:") {
		t.Errorf("%s: expected denial error, got: %v", desc, err)
	}
}

func exec_git_init(e *Executor, root string) {
	e.Run("git init")
}
