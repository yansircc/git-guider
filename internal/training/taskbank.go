package training

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Task struct {
	ID          string           `json:"id"`
	Difficulty  int              `json:"difficulty"`
	Description string           `json:"description"`
	Setup       []string         `json:"setup"`
	Verify      []map[string]any `json:"verify"`
	Hints       []string         `json:"hints"`
	OnPassNote  string           `json:"on_pass_note"`
	Chain       string           `json:"chain,omitempty"`
	ChainOrder  int              `json:"chain_order,omitempty"`
}

type TopicEntry struct {
	Name  string `json:"name"`
	Tasks []Task `json:"tasks"`
}

type TaskBank struct {
	Topics map[string]TopicEntry // keyed by "L1.1", "L2.1", etc.
}

func LoadTaskBank(path string) (*TaskBank, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("task bank not found: %w", err)
	}

	bank := &TaskBank{Topics: make(map[string]TopicEntry)}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			if err := bank.loadFile(filepath.Join(path, e.Name())); err != nil {
				return nil, fmt.Errorf("load %s: %w", e.Name(), err)
			}
		}
	} else {
		if err := bank.loadFile(path); err != nil {
			return nil, err
		}
	}

	return bank, nil
}

func (b *TaskBank) loadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var topics map[string]TopicEntry
	if err := json.Unmarshal(data, &topics); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	for k, v := range topics {
		b.Topics[k] = v
	}
	return nil
}

func (b *TaskBank) GetTask(taskID string) (*Task, string, error) {
	for topicKey, entry := range b.Topics {
		for i := range entry.Tasks {
			if entry.Tasks[i].ID == taskID {
				return &entry.Tasks[i], topicKey, nil
			}
		}
	}
	return nil, "", fmt.Errorf("task %s not found", taskID)
}

func (b *TaskBank) SortedTopicKeys() []string {
	keys := make([]string, 0, len(b.Topics))
	for k := range b.Topics {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
