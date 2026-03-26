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

// TestSetupFailureKeepsValidCWD verifies that if a setup command fails,
// the DB session still points to a valid (sandboxRoot) directory.
func TestSetupFailureKeepsValidCWD(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	sandboxBase := filepath.Join(tmpDir, "sandboxes")
	os.MkdirAll(sandboxBase, 0o755)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	// Build a task bank with a task whose setup will fail
	bank := &TaskBank{Topics: map[string]TopicEntry{
		"T1.1": {Name: "bad setup", Tasks: []Task{{
			ID:          "T1.1-bad",
			Difficulty:  1,
			Description: "task with bad setup",
			Setup: []string{
				"git init",
				"git checkout nonexistent-branch", // will fail
			},
			Verify: []map[string]any{{"type": "status_clean"}},
		}}},
	}}

	svc := NewService(store, bank, sandboxBase)

	sess, err := svc.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	defer sess.Destroy()

	// StartTask should fail on the bad setup command
	err = svc.StartTask(sess, "T1.1-bad")
	if err == nil {
		t.Fatal("expected StartTask to fail, but it succeeded")
	}

	// Reload session from DB — cwd must be valid (sandboxRoot, which exists)
	reloaded, err := store.LoadSession(sess.ID)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}

	// CWD should be sandboxRoot
	if reloaded.CWD != sess.SandboxRoot {
		t.Errorf("after setup failure, CWD=%q, want sandboxRoot=%q", reloaded.CWD, sess.SandboxRoot)
	}

	// TaskID should be empty (no active task)
	if reloaded.TaskID != "" {
		t.Errorf("after setup failure, TaskID=%q, want empty", reloaded.TaskID)
	}

	// The directory must actually exist
	if _, err := os.Stat(reloaded.CWD); err != nil {
		t.Errorf("CWD %q does not exist: %v", reloaded.CWD, err)
	}

	// An executor pointed at this CWD should work
	exec := cmdexec.NewExecutor(reloaded.SandboxRoot, reloaded.CWD)
	_, execErr := exec.Run("pwd")
	if execErr != nil {
		t.Errorf("executor at recovered CWD failed: %v", execErr)
	}
}
