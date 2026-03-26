package training

import (
	"os"
	"path/filepath"
	"testing"

	"git-guider/internal/cmdexec"
)

func setupService(t *testing.T) (*Service, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	sandboxBase := filepath.Join(tmpDir, "sandboxes")
	os.MkdirAll(sandboxBase, 0o755)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// Find tasks/vertical.json relative to project root
	taskPath := findVerticalJSON(t)
	bank, err := LoadTaskBank(taskPath)
	if err != nil {
		t.Fatalf("LoadTaskBank: %v", err)
	}

	svc := NewService(store, bank, sandboxBase)
	cleanup := func() {
		store.Close()
	}
	return svc, cleanup
}

func findVerticalJSON(t *testing.T) string {
	t.Helper()
	// Walk up from test file to find tasks/vertical.json
	dir, _ := os.Getwd()
	for {
		candidate := filepath.Join(dir, "tasks", "vertical.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("cannot find tasks/vertical.json")
		}
		dir = parent
	}
}

// TestVerticalSliceA: init + add + commit (L1)
func TestVerticalSliceA(t *testing.T) {
	svc, cleanup := setupService(t)
	defer cleanup()

	sess, err := svc.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	defer sess.Destroy()

	if err := svc.StartTask(sess, "L1.1-a"); err != nil {
		t.Fatalf("StartTask: %v", err)
	}

	exec := cmdexec.NewExecutor(sess.SandboxRoot, sess.CWD)
	cmds := []string{
		"git init",
		"echo hello > hello.txt",
		"git add .",
		"git commit -m 'first commit'",
	}
	for _, cmd := range cmds {
		result, err := exec.Run(cmd)
		if err != nil {
			t.Fatalf("exec %q: %v", cmd, err)
		}
		sess.CWD = result.CWD
	}

	vr, err := svc.VerifyTask(sess)
	if err != nil {
		t.Fatalf("VerifyTask: %v", err)
	}
	if !vr.Passed {
		for _, r := range vr.Results {
			if !r.Passed {
				t.Errorf("assertion failed: expected=%s actual=%s error=%s", r.Expected, r.Actual, r.Error)
			}
		}
		t.Fatal("task A did not pass")
	}
}

// TestVerticalSliceB: branch + merge (L2)
func TestVerticalSliceB(t *testing.T) {
	svc, cleanup := setupService(t)
	defer cleanup()

	sess, err := svc.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	defer sess.Destroy()

	if err := svc.StartTask(sess, "L2.1-a"); err != nil {
		t.Fatalf("StartTask: %v", err)
	}

	exec := cmdexec.NewExecutor(sess.SandboxRoot, sess.CWD)
	cmds := []string{
		"git checkout -b feature",
		"echo feature > feature.txt",
		"git add .",
		"git commit -m 'feature work'",
		"git checkout main",
		"git merge --no-ff feature -m 'merge feature'",
	}
	for _, cmd := range cmds {
		result, err := exec.Run(cmd)
		if err != nil {
			t.Fatalf("exec %q: %v", cmd, err)
		}
		sess.CWD = result.CWD
	}

	vr, err := svc.VerifyTask(sess)
	if err != nil {
		t.Fatalf("VerifyTask: %v", err)
	}
	if !vr.Passed {
		for _, r := range vr.Results {
			if !r.Passed {
				t.Errorf("assertion failed: expected=%s actual=%s error=%s", r.Expected, r.Actual, r.Error)
			}
		}
		t.Fatal("task B did not pass")
	}
}

// TestVerticalSliceC: clone + push (L3)
func TestVerticalSliceC(t *testing.T) {
	svc, cleanup := setupService(t)
	defer cleanup()

	sess, err := svc.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	defer sess.Destroy()

	if err := svc.StartTask(sess, "L3.1-a"); err != nil {
		t.Fatalf("StartTask: %v", err)
	}

	// User actions: add a file and push
	exec := cmdexec.NewExecutor(sess.SandboxRoot, sess.CWD)
	cmds := []string{
		"echo new > new.txt",
		"git add .",
		"git commit -m 'add new'",
		"git push",
	}
	for _, cmd := range cmds {
		result, err := exec.Run(cmd)
		if err != nil {
			t.Fatalf("exec %q: %v", cmd, err)
		}
		sess.CWD = result.CWD
	}

	vr, err := svc.VerifyTask(sess)
	if err != nil {
		t.Fatalf("VerifyTask: %v", err)
	}
	if !vr.Passed {
		for _, r := range vr.Results {
			if !r.Passed {
				t.Errorf("assertion failed: expected=%s actual=%s error=%s", r.Expected, r.Actual, r.Error)
			}
		}
		t.Fatal("task C did not pass")
	}
}

// TestVerticalSliceD: worktree (L4)
func TestVerticalSliceD(t *testing.T) {
	svc, cleanup := setupService(t)
	defer cleanup()

	sess, err := svc.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	defer sess.Destroy()

	if err := svc.StartTask(sess, "L4.1-a"); err != nil {
		t.Fatalf("StartTask: %v", err)
	}

	exec := cmdexec.NewExecutor(sess.SandboxRoot, sess.CWD)
	cmds := []string{
		"git worktree add ../wt feature",
	}
	for _, cmd := range cmds {
		result, err := exec.Run(cmd)
		if err != nil {
			t.Fatalf("exec %q: %v", cmd, err)
		}
		sess.CWD = result.CWD
	}

	vr, err := svc.VerifyTask(sess)
	if err != nil {
		t.Fatalf("VerifyTask: %v", err)
	}
	if !vr.Passed {
		for _, r := range vr.Results {
			if !r.Passed {
				t.Errorf("assertion failed: expected=%s actual=%s error=%s", r.Expected, r.Actual, r.Error)
			}
		}
		t.Fatal("task D did not pass")
	}
}
