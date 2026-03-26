package training

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"git-guider/internal/cmdexec"
	"git-guider/internal/session"
	"git-guider/internal/verify"
)

type Service struct {
	store       *Store
	taskBank    *TaskBank
	sandboxBase string
}

type VerifyResult struct {
	Passed     bool            `json:"passed"`
	Results    []verify.Result `json:"results"`
	Hints      []string        `json:"hints,omitempty"`
	OnPassNote string          `json:"on_pass_note,omitempty"`
	Duration   int             `json:"duration_sec"`
}

func NewService(store *Store, taskBank *TaskBank, sandboxBase string) *Service {
	return &Service{
		store:       store,
		taskBank:    taskBank,
		sandboxBase: sandboxBase,
	}
}

func (s *Service) CreateSession() (*session.Session, error) {
	sess, err := session.New(s.sandboxBase)
	if err != nil {
		return nil, err
	}
	if err := s.store.SaveSession(sess); err != nil {
		sess.Destroy()
		return nil, err
	}
	return sess, nil
}

func (s *Service) GetSession(id string) (*session.Session, error) {
	return s.store.LoadSession(id)
}

func (s *Service) SelectNextTask(sessionID string) (*Task, error) {
	task, _, err := s.store.SelectNextTask(s.taskBank)
	return task, err
}

func (s *Service) StartTask(sess *session.Session, taskID string) error {
	task, _, err := s.taskBank.GetTask(taskID)
	if err != nil {
		return err
	}

	// Persist baseline BEFORE cleaning: cwd=sandboxRoot (always valid since
	// cleanSandbox only removes contents, not the root itself), no active task.
	// This ensures the DB never holds a cwd that clean might have deleted.
	sess.CWD = sess.SandboxRoot
	sess.TaskID = ""
	sess.TaskJSON = ""
	if err := s.store.SaveSession(sess); err != nil {
		return fmt.Errorf("save baseline session: %w", err)
	}

	// Clean the sandbox: remove all contents, propagate errors.
	// Even if this fails partway, DB cwd=sandboxRoot is still valid.
	if err := cleanSandbox(sess.SandboxRoot); err != nil {
		return fmt.Errorf("clean sandbox: %w", err)
	}

	// Run setup commands through the unified executor
	exec := cmdexec.NewExecutor(sess.SandboxRoot, sess.CWD)
	for _, cmd := range task.Setup {
		result, err := exec.Run(cmd)
		if err != nil {
			// Setup failed. DB already has cwd=sandboxRoot (valid).
			return fmt.Errorf("setup command %q: %w", cmd, err)
		}
		sess.CWD = result.CWD
		exec.CWD = result.CWD
	}

	// Setup succeeded — now persist the final state with the active task.
	sess.TaskID = taskID
	sess.TaskJSON = taskToJSON(task)
	sess.IssuedAt = time.Now()
	return s.store.SaveSession(sess)
}

func cleanSandbox(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() == ".gitconfig" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(root, e.Name())); err != nil {
			return fmt.Errorf("clean %s: %w", e.Name(), err)
		}
	}
	return nil
}

func (s *Service) VerifyTask(sess *session.Session) (*VerifyResult, error) {
	if sess.TaskID == "" {
		return nil, fmt.Errorf("no active task")
	}

	var task Task
	if err := json.Unmarshal([]byte(sess.TaskJSON), &task); err != nil {
		return nil, fmt.Errorf("parse task: %w", err)
	}

	_, topicKey, err := s.taskBank.GetTask(sess.TaskID)
	if err != nil {
		return nil, err
	}

	vr := &VerifyResult{
		Passed: true,
	}

	for _, rawCheck := range task.Verify {
		assertion := verify.ParseAssertion(rawCheck)
		result := verify.Evaluate(assertion, sess.SandboxRoot, sess.CWD)
		vr.Results = append(vr.Results, result)
		if !result.Passed {
			vr.Passed = false
		}
	}

	duration := int(time.Since(sess.IssuedAt).Seconds())
	vr.Duration = duration

	if !vr.Passed {
		vr.Hints = task.Hints
	} else {
		vr.OnPassNote = task.OnPassNote
	}

	// Record attempt
	errorDetail := ""
	if !vr.Passed {
		detailBytes, _ := json.Marshal(vr.Results)
		errorDetail = string(detailBytes)
	}
	s.store.RecordAttempt(sess.ID, topicKey, sess.TaskID, task.Difficulty, vr.Passed, errorDetail, duration)
	s.store.UpdateMastery(topicKey, vr.Passed)

	// Clear task on pass
	if vr.Passed {
		sess.TaskID = ""
		sess.TaskJSON = ""
		s.store.SaveSession(sess)
	}

	return vr, nil
}

func (s *Service) GetProgress() (*Progress, error) {
	return s.store.GetProgress(s.taskBank)
}

func (s *Service) GetExecutor(sess *session.Session) *cmdexec.Executor {
	return cmdexec.NewExecutor(sess.SandboxRoot, sess.CWD)
}

// SessionResponse is the JSON-friendly view of a session for the API.
// Hides internal fields (SandboxRoot, TaskJSON).
type SessionResponse struct {
	ID     string `json:"id"`
	TaskID string `json:"task_id"`
	CWD    string `json:"cwd"` // display-only relative path
}

func (s *Service) SessionToResponse(sess *session.Session) SessionResponse {
	return SessionResponse{
		ID:     sess.ID,
		TaskID: sess.TaskID,
		CWD:    sess.RelativeCWD(),
	}
}

// SessionInfo keeps the full internal session for server-side use.
type SessionInfo = session.Session

func (s *Service) UpdateSessionCWD(sessionID, cwd string) {
	sess, err := s.store.LoadSession(sessionID)
	if err != nil {
		return
	}
	sess.CWD = cwd
	s.store.SaveSession(sess)
}

type LevelInfo struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Tasks  []Task `json:"tasks"`
}

func (s *Service) GetLevels() []LevelInfo {
	var levels []LevelInfo
	for _, key := range s.taskBank.SortedTopicKeys() {
		entry := s.taskBank.Topics[key]
		levels = append(levels, LevelInfo{
			Key:   key,
			Name:  entry.Name,
			Tasks: entry.Tasks,
		})
	}
	return levels
}
