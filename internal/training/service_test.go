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
	dir, _ := os.Getwd()
	// testdata is in the same package directory
	candidate := filepath.Join(dir, "testdata", "vertical.json")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	t.Fatalf("cannot find testdata/vertical.json from %s", dir)
	return ""
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

func TestCleanSandboxRemovesAllAndPreservesGitconfig(t *testing.T) {
	root := t.TempDir()

	// Create some files and dirs that should be cleaned
	os.WriteFile(filepath.Join(root, ".gitconfig"), []byte("[user]\n"), 0o644)
	os.WriteFile(filepath.Join(root, "file.txt"), []byte("data"), 0o644)
	os.MkdirAll(filepath.Join(root, "subdir", "nested"), 0o755)
	os.WriteFile(filepath.Join(root, "subdir", "nested", "deep.txt"), []byte("deep"), 0o644)
	os.MkdirAll(filepath.Join(root, ".git", "objects"), 0o755)

	if err := cleanSandbox(root); err != nil {
		t.Fatalf("cleanSandbox: %v", err)
	}

	entries, _ := os.ReadDir(root)
	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
	}

	// .gitconfig must survive
	if !names[".gitconfig"] {
		t.Error(".gitconfig was deleted, should be preserved")
	}
	// everything else must be gone
	for name := range names {
		if name != ".gitconfig" {
			t.Errorf("entry %q survived cleanSandbox, should be deleted", name)
		}
	}
}

// TestStartTaskBaselineSavedBeforeClean verifies that even if the old CWD
// was a subdirectory that cleanSandbox deletes, the DB session already
// points to sandboxRoot (which always exists) before clean runs.
func TestStartTaskBaselineSavedBeforeClean(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	sandboxBase := filepath.Join(tmpDir, "sandboxes")
	os.MkdirAll(sandboxBase, 0o755)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	bank := &TaskBank{Topics: map[string]TopicEntry{
		"T1.1": {Name: "simple", Tasks: []Task{{
			ID:          "T1.1-a",
			Difficulty:  1,
			Description: "simple task",
			Setup:       []string{},
			Verify:      []map[string]any{{"type": "status_clean"}},
		}}},
	}}

	svc := NewService(store, bank, sandboxBase)

	sess, err := svc.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	defer sess.Destroy()

	// Simulate a previous task that left CWD inside a subdirectory
	subdir := filepath.Join(sess.SandboxRoot, "old-workspace")
	os.MkdirAll(subdir, 0o755)
	sess.CWD = subdir
	sess.TaskID = "old-task"
	store.SaveSession(sess)

	// Confirm the subdir exists before StartTask
	if _, err := os.Stat(subdir); err != nil {
		t.Fatalf("subdir should exist before StartTask: %v", err)
	}

	// StartTask will save baseline (cwd=sandboxRoot) then clean (deleting subdir)
	if err := svc.StartTask(sess, "T1.1-a"); err != nil {
		t.Fatalf("StartTask: %v", err)
	}

	// The old subdir should be gone
	if _, err := os.Stat(subdir); err == nil {
		t.Error("old subdir should have been cleaned")
	}

	// DB session must point to sandboxRoot, not the deleted subdir
	reloaded, err := store.LoadSession(sess.ID)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}

	// After successful StartTask, CWD is sandboxRoot (setup was empty)
	if reloaded.CWD != sess.SandboxRoot {
		t.Errorf("CWD=%q, want sandboxRoot=%q", reloaded.CWD, sess.SandboxRoot)
	}

	// The CWD directory must exist
	if _, err := os.Stat(reloaded.CWD); err != nil {
		t.Errorf("CWD %q does not exist: %v", reloaded.CWD, err)
	}
}
