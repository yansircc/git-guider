package verify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Assertion struct {
	Type   string         `json:"type"`
	Params map[string]any `json:"-"`
	Repo   string         `json:"repo,omitempty"`
}

type Result struct {
	Passed   bool
	Expected string
	Actual   string
	Error    string
}

func Evaluate(a Assertion, sandboxRoot, cwd string) Result {
	repoDir := cwd
	if a.Repo != "" {
		repoDir = filepath.Join(sandboxRoot, a.Repo)
	}

	switch a.Type {
	case "branch_exists":
		return checkBranchExists(repoDir, paramStr(a.Params, "name"))
	case "branch_current":
		return checkBranchCurrent(repoDir, paramStr(a.Params, "name"))
	case "commit_count":
		return checkCommitCount(repoDir, paramInt(a.Params, "min"), paramStr(a.Params, "ref"))
	case "commit_message_contains":
		return checkCommitMessageContains(repoDir, paramStr(a.Params, "pattern"), paramStr(a.Params, "ref"))
	case "file_exists":
		return checkFileExists(cwd, paramStr(a.Params, "path"))
	case "file_not_exists":
		return checkFileNotExists(cwd, paramStr(a.Params, "path"))
	case "file_contains":
		return checkFileContains(cwd, paramStr(a.Params, "path"), paramStr(a.Params, "pattern"))
	case "file_staged":
		return checkFileStaged(repoDir, paramStr(a.Params, "path"))
	case "status_clean":
		return checkStatusClean(repoDir)
	case "tag_exists":
		return checkTagExists(repoDir, paramStr(a.Params, "name"))
	case "remote_exists":
		return checkRemoteExists(repoDir, paramStr(a.Params, "name"))
	case "remote_url":
		return checkRemoteURL(repoDir, paramStr(a.Params, "name"), paramStr(a.Params, "url"))
	case "stash_count":
		return checkStashCount(repoDir, paramInt(a.Params, "min"))
	case "worktree_count":
		return checkWorktreeCount(repoDir, paramInt(a.Params, "min"))
	case "config_equals":
		return checkConfigEquals(repoDir, paramStr(a.Params, "key"), paramStr(a.Params, "value"))
	case "ref_equals":
		return checkRefEquals(repoDir, paramStr(a.Params, "ref_a"), paramStr(a.Params, "ref_b"))
	case "gitignore_ignores":
		return checkGitignoreIgnores(repoDir, paramStr(a.Params, "path"))
	case "merge_clean":
		return checkMergeClean(repoDir)
	case "file_count":
		return checkFileCount(cwd, paramStr(a.Params, "pattern"), paramInt(a.Params, "min"))

	// DAG assertions
	case "head_parents":
		return checkHeadParents(repoDir, paramInt(a.Params, "count"))
	case "ref_ancestor_of":
		return checkRefAncestorOf(repoDir, paramStr(a.Params, "ancestor"), paramStr(a.Params, "descendant"))
	case "ref_not_ancestor_of":
		return checkRefNotAncestorOf(repoDir, paramStr(a.Params, "ref_a"), paramStr(a.Params, "ref_b"))
	case "commit_tree_matches":
		return checkCommitTreeMatches(repoDir, paramStr(a.Params, "ref"), paramStr(a.Params, "tree_ref"))
	case "commit_parents":
		return checkCommitParents(repoDir, paramStr(a.Params, "ref"), paramStrSlice(a.Params, "parents"))
	case "commit_is_not":
		return checkCommitIsNot(repoDir, paramStr(a.Params, "ref"), paramStr(a.Params, "forbidden_ref"))
	case "linear_history":
		return checkLinearHistory(repoDir, paramStr(a.Params, "ref"), paramInt(a.Params, "count"))
	case "ref_points_to":
		return checkRefPointsTo(repoDir, paramStr(a.Params, "ref"), paramStr(a.Params, "target_ref"))

	default:
		return Result{Error: fmt.Sprintf("unknown assertion type: %s", a.Type)}
	}
}

// --- helpers ---

func git(repoDir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir
	cmd.Env = []string{
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_TERMINAL_PROMPT=0",
		"HOME=" + repoDir,
		"PATH=" + os.Getenv("PATH"),
	}
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func paramStr(p map[string]any, key string) string {
	v, ok := p[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func paramInt(p map[string]any, key string) int {
	v, ok := p[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case string:
		i, _ := strconv.Atoi(n)
		return i
	}
	return 0
}

func paramStrSlice(p map[string]any, key string) []string {
	v, ok := p[key]
	if !ok {
		return nil
	}
	switch s := v.(type) {
	case []any:
		result := make([]string, len(s))
		for i, item := range s {
			result[i], _ = item.(string)
		}
		return result
	case []string:
		return s
	}
	return nil
}

func pass() Result   { return Result{Passed: true} }
func fail(expected, actual string) Result {
	return Result{Passed: false, Expected: expected, Actual: actual}
}

// --- basic assertions ---

func checkBranchExists(repoDir, name string) Result {
	out, err := git(repoDir, "branch", "--list", name)
	if err != nil {
		return Result{Error: err.Error()}
	}
	if strings.TrimSpace(out) == "" {
		return fail("branch "+name+" exists", "not found")
	}
	return pass()
}

func checkBranchCurrent(repoDir, name string) Result {
	out, err := git(repoDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return Result{Error: err.Error()}
	}
	if out != name {
		return fail("current branch: "+name, out)
	}
	return pass()
}

func checkCommitCount(repoDir string, min int, ref string) Result {
	if ref == "" {
		ref = "HEAD"
	}
	out, err := git(repoDir, "rev-list", "--count", ref)
	if err != nil {
		return Result{Error: err.Error()}
	}
	count, _ := strconv.Atoi(out)
	if count < min {
		return fail(fmt.Sprintf(">= %d commits", min), fmt.Sprintf("%d commits", count))
	}
	return pass()
}

func checkCommitMessageContains(repoDir, pattern, ref string) Result {
	if ref == "" {
		ref = "HEAD"
	}
	out, err := git(repoDir, "log", "-1", "--format=%s", ref)
	if err != nil {
		return Result{Error: err.Error()}
	}
	if !strings.Contains(out, pattern) {
		return fail("commit message contains: "+pattern, out)
	}
	return pass()
}

func checkFileExists(cwd, path string) Result {
	target := filepath.Join(cwd, path)
	if _, err := os.Stat(target); err != nil {
		return fail("file exists: "+path, "not found")
	}
	return pass()
}

func checkFileNotExists(cwd, path string) Result {
	target := filepath.Join(cwd, path)
	if _, err := os.Stat(target); err == nil {
		return fail("file not exists: "+path, "exists")
	}
	return pass()
}

func checkFileContains(cwd, path, pattern string) Result {
	target := filepath.Join(cwd, path)
	data, err := os.ReadFile(target)
	if err != nil {
		return Result{Error: err.Error()}
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return Result{Error: "invalid pattern: " + err.Error()}
	}
	if !re.Match(data) {
		return fail("file matches: "+pattern, string(data))
	}
	return pass()
}

func checkFileStaged(repoDir, path string) Result {
	out, err := git(repoDir, "diff", "--cached", "--name-only")
	if err != nil {
		return Result{Error: err.Error()}
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == path {
			return pass()
		}
	}
	return fail("file staged: "+path, "not staged")
}

func checkStatusClean(repoDir string) Result {
	out, err := git(repoDir, "status", "--porcelain")
	if err != nil {
		return Result{Error: err.Error()}
	}
	if strings.TrimSpace(out) != "" {
		return fail("clean status", out)
	}
	return pass()
}

func checkTagExists(repoDir, name string) Result {
	out, err := git(repoDir, "tag", "--list", name)
	if err != nil {
		return Result{Error: err.Error()}
	}
	if strings.TrimSpace(out) == "" {
		return fail("tag "+name+" exists", "not found")
	}
	return pass()
}

func checkRemoteExists(repoDir, name string) Result {
	out, err := git(repoDir, "remote")
	if err != nil {
		return Result{Error: err.Error()}
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == name {
			return pass()
		}
	}
	return fail("remote "+name+" exists", "not found")
}

func checkRemoteURL(repoDir, name, url string) Result {
	out, err := git(repoDir, "remote", "get-url", name)
	if err != nil {
		return Result{Error: err.Error()}
	}
	if out != url {
		return fail("remote "+name+" url: "+url, out)
	}
	return pass()
}

func checkStashCount(repoDir string, min int) Result {
	out, _ := git(repoDir, "stash", "list")
	count := 0
	if strings.TrimSpace(out) != "" {
		count = len(strings.Split(out, "\n"))
	}
	if count < min {
		return fail(fmt.Sprintf(">= %d stashes", min), fmt.Sprintf("%d stashes", count))
	}
	return pass()
}

func checkWorktreeCount(repoDir string, min int) Result {
	out, err := git(repoDir, "worktree", "list")
	if err != nil {
		return Result{Error: err.Error()}
	}
	count := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	if count < min {
		return fail(fmt.Sprintf(">= %d worktrees", min), fmt.Sprintf("%d worktrees", count))
	}
	return pass()
}

func checkConfigEquals(repoDir, key, value string) Result {
	out, err := git(repoDir, "config", "--get", key)
	if err != nil {
		return Result{Error: fmt.Sprintf("git config --get %s: %v", key, err)}
	}
	if out != value {
		return fail(key+"="+value, key+"="+out)
	}
	return pass()
}

func checkRefEquals(repoDir, refA, refB string) Result {
	a, err := git(repoDir, "rev-parse", refA)
	if err != nil {
		return Result{Error: fmt.Sprintf("rev-parse %s: %v", refA, err)}
	}
	b, err := git(repoDir, "rev-parse", refB)
	if err != nil {
		return Result{Error: fmt.Sprintf("rev-parse %s: %v", refB, err)}
	}
	if a != b {
		return fail(refA+" == "+refB, fmt.Sprintf("%s != %s", a[:8], b[:8]))
	}
	return pass()
}

func checkGitignoreIgnores(repoDir, path string) Result {
	_, err := git(repoDir, "check-ignore", path)
	if err != nil {
		return fail("gitignore ignores: "+path, "not ignored")
	}
	return pass()
}

func checkMergeClean(repoDir string) Result {
	mergeHead := filepath.Join(repoDir, ".git", "MERGE_HEAD")
	if _, err := os.Stat(mergeHead); err == nil {
		return fail("no active merge", "MERGE_HEAD exists")
	}
	return pass()
}

func checkFileCount(cwd, pattern string, min int) Result {
	matches, err := filepath.Glob(filepath.Join(cwd, pattern))
	if err != nil {
		return Result{Error: err.Error()}
	}
	if len(matches) < min {
		return fail(fmt.Sprintf(">= %d files matching %s", min, pattern), fmt.Sprintf("%d files", len(matches)))
	}
	return pass()
}

// --- DAG assertions ---

func checkHeadParents(repoDir string, count int) Result {
	out, err := git(repoDir, "rev-parse", "HEAD^@")
	if err != nil {
		// If HEAD has no parents (initial commit), rev-parse HEAD^@ may fail
		if count == 0 {
			return pass()
		}
		return Result{Error: err.Error()}
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	actual := 0
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			actual++
		}
	}
	if actual != count {
		return fail(fmt.Sprintf("HEAD has %d parents", count), fmt.Sprintf("%d parents", actual))
	}
	return pass()
}

func checkRefAncestorOf(repoDir, ancestor, descendant string) Result {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", ancestor, descendant)
	cmd.Dir = repoDir
	cmd.Env = []string{
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"HOME=" + repoDir, "PATH=" + os.Getenv("PATH"),
	}
	if err := cmd.Run(); err != nil {
		return fail(ancestor+" is ancestor of "+descendant, "not an ancestor")
	}
	return pass()
}

func checkRefNotAncestorOf(repoDir, refA, refB string) Result {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", refA, refB)
	cmd.Dir = repoDir
	cmd.Env = []string{
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"HOME=" + repoDir, "PATH=" + os.Getenv("PATH"),
	}
	if err := cmd.Run(); err != nil {
		return pass() // not an ancestor => pass
	}
	return fail(refA+" is NOT ancestor of "+refB, "is an ancestor")
}

func checkCommitTreeMatches(repoDir, ref, treeRef string) Result {
	treeA, err := git(repoDir, "rev-parse", ref+"^{tree}")
	if err != nil {
		return Result{Error: fmt.Sprintf("rev-parse %s^{tree}: %v", ref, err)}
	}
	treeB, err := git(repoDir, "rev-parse", treeRef+"^{tree}")
	if err != nil {
		return Result{Error: fmt.Sprintf("rev-parse %s^{tree}: %v", treeRef, err)}
	}
	if treeA != treeB {
		return fail("trees match", fmt.Sprintf("%s != %s", treeA[:8], treeB[:8]))
	}
	return pass()
}

func checkCommitParents(repoDir, ref string, parents []string) Result {
	out, err := git(repoDir, "rev-parse", ref+"^@")
	if err != nil && len(parents) > 0 {
		return Result{Error: err.Error()}
	}
	var actual []string
	if strings.TrimSpace(out) != "" {
		for _, l := range strings.Split(out, "\n") {
			if t := strings.TrimSpace(l); t != "" {
				actual = append(actual, t)
			}
		}
	}
	if len(actual) != len(parents) {
		return fail(fmt.Sprintf("%d parents", len(parents)), fmt.Sprintf("%d parents", len(actual)))
	}
	for i, expected := range parents {
		resolved, err := git(repoDir, "rev-parse", expected)
		if err != nil {
			return Result{Error: fmt.Sprintf("rev-parse %s: %v", expected, err)}
		}
		if actual[i] != resolved {
			return fail(expected, actual[i][:8])
		}
	}
	return pass()
}

func checkCommitIsNot(repoDir, ref, forbiddenRef string) Result {
	a, err := git(repoDir, "rev-parse", ref)
	if err != nil {
		return Result{Error: fmt.Sprintf("rev-parse %s: %v", ref, err)}
	}
	b, err := git(repoDir, "rev-parse", forbiddenRef)
	if err != nil {
		return Result{Error: fmt.Sprintf("rev-parse %s: %v", forbiddenRef, err)}
	}
	if a == b {
		return fail(ref+" != "+forbiddenRef, "same commit: "+a[:8])
	}
	return pass()
}

func checkLinearHistory(repoDir, ref string, count int) Result {
	if ref == "" {
		ref = "HEAD"
	}
	// Check no merge commits
	out, _ := git(repoDir, "rev-list", "--merges", ref, "-n1")
	if strings.TrimSpace(out) != "" {
		return fail("linear history (no merges)", "merge commit found: "+out[:8])
	}
	// Check commit count
	if count > 0 {
		countOut, err := git(repoDir, "rev-list", "--count", ref)
		if err != nil {
			return Result{Error: err.Error()}
		}
		actual, _ := strconv.Atoi(countOut)
		if actual < count {
			return fail(fmt.Sprintf(">= %d commits", count), fmt.Sprintf("%d commits", actual))
		}
	}
	return pass()
}

func checkRefPointsTo(repoDir, ref, targetRef string) Result {
	a, err := git(repoDir, "rev-parse", ref)
	if err != nil {
		return Result{Error: fmt.Sprintf("rev-parse %s: %v", ref, err)}
	}
	b, err := git(repoDir, "rev-parse", targetRef)
	if err != nil {
		return Result{Error: fmt.Sprintf("rev-parse %s: %v", targetRef, err)}
	}
	if a != b {
		return fail(ref+" -> "+targetRef, fmt.Sprintf("%s != %s", a[:8], b[:8]))
	}
	return pass()
}

// ParseAssertion converts a raw JSON map into an Assertion with typed Params.
func ParseAssertion(raw map[string]any) Assertion {
	a := Assertion{
		Params: make(map[string]any),
	}
	for k, v := range raw {
		switch k {
		case "type":
			a.Type, _ = v.(string)
		case "repo":
			a.Repo, _ = v.(string)
		default:
			a.Params[k] = v
		}
	}
	return a
}
