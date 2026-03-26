package training

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"git-guider/internal/session"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			sandbox_root TEXT NOT NULL,
			cwd TEXT NOT NULL,
			task_id TEXT NOT NULL DEFAULT '',
			task_json TEXT NOT NULL DEFAULT '',
			issued_at TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS attempts (
			id INTEGER PRIMARY KEY,
			session_id TEXT NOT NULL,
			topic TEXT NOT NULL,
			task_id TEXT NOT NULL,
			difficulty INTEGER,
			passed INTEGER NOT NULL,
			error_detail TEXT,
			duration_sec INTEGER,
			created_at TEXT DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS topic_mastery (
			topic TEXT PRIMARY KEY,
			consecutive_passes INTEGER DEFAULT 0,
			mastered INTEGER DEFAULT 0,
			mastered_at TEXT,
			updated_at TEXT DEFAULT (datetime('now'))
		);
	`)
	return err
}

func (s *Store) SaveSession(sess *session.Session) error {
	_, err := s.db.Exec(`
		INSERT INTO sessions (id, sandbox_root, cwd, task_id, task_json, issued_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			cwd = excluded.cwd,
			task_id = excluded.task_id,
			task_json = excluded.task_json,
			issued_at = excluded.issued_at
	`, sess.ID, sess.SandboxRoot, sess.CWD, sess.TaskID, sess.TaskJSON,
		sess.IssuedAt.Format(time.RFC3339), sess.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *Store) LoadSession(id string) (*session.Session, error) {
	var sess session.Session
	var issuedAt, createdAt string
	err := s.db.QueryRow(`
		SELECT id, sandbox_root, cwd, task_id, task_json, issued_at, created_at
		FROM sessions WHERE id = ?
	`, id).Scan(&sess.ID, &sess.SandboxRoot, &sess.CWD, &sess.TaskID,
		&sess.TaskJSON, &issuedAt, &createdAt)
	if err != nil {
		return nil, err
	}
	sess.IssuedAt, _ = time.Parse(time.RFC3339, issuedAt)
	sess.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &sess, nil
}

func (s *Store) RecordAttempt(sessionID, topic, taskID string, difficulty int, passed bool, errorDetail string, durationSec int) error {
	passedInt := 0
	if passed {
		passedInt = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO attempts (session_id, topic, task_id, difficulty, passed, error_detail, duration_sec)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, sessionID, topic, taskID, difficulty, passedInt, errorDetail, durationSec)
	return err
}

func (s *Store) ResetAll() error {
	_, err := s.db.Exec(`
		DELETE FROM attempts;
		DELETE FROM topic_mastery;
	`)
	return err
}

func (s *Store) UpdateMastery(topic string, passed bool) error {
	if passed {
		_, err := s.db.Exec(`
			INSERT INTO topic_mastery (topic, consecutive_passes, mastered, updated_at)
			VALUES (?, 1, 0, datetime('now'))
			ON CONFLICT(topic) DO UPDATE SET
				consecutive_passes = consecutive_passes + 1,
				mastered = CASE WHEN consecutive_passes + 1 >= 1 THEN 1 ELSE mastered END,
				mastered_at = CASE WHEN consecutive_passes + 1 >= 1 THEN datetime('now') ELSE mastered_at END,
				updated_at = datetime('now')
		`, topic)
		return err
	}

	_, err := s.db.Exec(`
		INSERT INTO topic_mastery (topic, consecutive_passes, mastered, updated_at)
		VALUES (?, 0, 0, datetime('now'))
		ON CONFLICT(topic) DO UPDATE SET
			consecutive_passes = 0,
			updated_at = datetime('now')
	`, topic)
	return err
}

func (s *Store) IsMastered(topic string) (bool, error) {
	var mastered int
	err := s.db.QueryRow(`SELECT mastered FROM topic_mastery WHERE topic = ?`, topic).Scan(&mastered)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return mastered == 1, nil
}

type Progress struct {
	Layers          []LayerProgress `json:"layers"`
	TotalTopics     int             `json:"total_topics"`
	TotalMastered   int             `json:"total_mastered"`
	TotalAttempts   int             `json:"total_attempts"`
	TotalPasses     int             `json:"total_passes"`
	PassRate        float64         `json:"pass_rate"`
	MasteredTopics  []string        `json:"mastered_topics"`
}

type LayerProgress struct {
	Layer    string `json:"layer"`
	Mastered int    `json:"mastered"`
	Total    int    `json:"total"`
}

func (s *Store) GetProgress(bank *TaskBank) (*Progress, error) {
	p := &Progress{}
	layers := map[string]*LayerProgress{}

	for _, key := range bank.SortedTopicKeys() {
		layer := string(key[1]) // "L1.1" → "1"
		if _, ok := layers[layer]; !ok {
			layers[layer] = &LayerProgress{Layer: layer}
		}
		layers[layer].Total++
		p.TotalTopics++

		mastered, _ := s.IsMastered(key)
		if mastered {
			layers[layer].Mastered++
			p.TotalMastered++
			p.MasteredTopics = append(p.MasteredTopics, key)
		}
	}

	for _, lp := range layers {
		p.Layers = append(p.Layers, *lp)
	}

	var totalAttempts, totalPasses int
	s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(passed), 0) FROM attempts`).Scan(&totalAttempts, &totalPasses)
	p.TotalAttempts = totalAttempts
	p.TotalPasses = totalPasses
	if totalAttempts > 0 {
		p.PassRate = float64(totalPasses) / float64(totalAttempts)
	}

	return p, nil
}

// SelectNextTask picks the next unmastered task.
func (s *Store) SelectNextTask(bank *TaskBank) (*Task, string, error) {
	for _, topicKey := range bank.SortedTopicKeys() {
		mastered, _ := s.IsMastered(topicKey)
		if mastered {
			continue
		}
		entry := bank.Topics[topicKey]
		task := s.pickLeastAttempted(entry.Tasks)
		if task != nil {
			return task, topicKey, nil
		}
	}
	return nil, "", fmt.Errorf("all topics mastered")
}

func (s *Store) pickLeastAttempted(tasks []Task) *Task {
	if len(tasks) == 0 {
		return nil
	}

	// Chain tasks: find first not-passed in chain order
	type chainTask struct {
		task  *Task
		order int
	}
	chains := map[string][]chainTask{}
	var unchained []*Task

	for i := range tasks {
		t := &tasks[i]
		if t.Chain != "" {
			chains[t.Chain] = append(chains[t.Chain], chainTask{task: t, order: t.ChainOrder})
		} else {
			unchained = append(unchained, t)
		}
	}

	// Try chain tasks first
	for _, ctasks := range chains {
		sort.Slice(ctasks, func(i, j int) bool { return ctasks[i].order < ctasks[j].order })
		for _, ct := range ctasks {
			var passed int
			s.db.QueryRow(`SELECT COUNT(*) FROM attempts WHERE task_id = ? AND passed = 1`, ct.task.ID).Scan(&passed)
			if passed == 0 {
				return ct.task
			}
		}
	}

	// Unchained: pick least attempted
	if len(unchained) == 0 {
		return nil
	}
	best := unchained[0]
	bestCount := s.attemptCount(best.ID)
	for _, t := range unchained[1:] {
		c := s.attemptCount(t.ID)
		if c < bestCount {
			best = t
			bestCount = c
		}
	}
	return best
}

func (s *Store) attemptCount(taskID string) int {
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM attempts WHERE task_id = ?`, taskID).Scan(&count)
	return count
}

func taskToJSON(t *Task) string {
	data, _ := json.Marshal(t)
	return string(data)
}
