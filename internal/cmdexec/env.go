package cmdexec

import (
	"os"
	"path/filepath"
)

// EnsureGitConfig writes a controlled .gitconfig in sandboxRoot.
// Must be called once per sandbox before any git commands.
func EnsureGitConfig(sandboxRoot string) error {
	configPath := filepath.Join(sandboxRoot, ".gitconfig")
	content := `[init]
	defaultBranch = main
[user]
	name = Learner
	email = learner@git-guider
`
	return os.WriteFile(configPath, []byte(content), 0o644)
}

func BaselineEnv(sandboxRoot string) []string {
	return []string{
		"HOME=" + sandboxRoot,
		"GIT_CONFIG_GLOBAL=" + filepath.Join(sandboxRoot, ".gitconfig"),
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_AUTHOR_NAME=Learner",
		"GIT_AUTHOR_EMAIL=learner@git-guider",
		"GIT_COMMITTER_NAME=Learner",
		"GIT_COMMITTER_EMAIL=learner@git-guider",
		"LANG=en_US.UTF-8",
		"PATH=" + os.Getenv("PATH"),
	}
}
