package cmdexec

import (
	"fmt"
	"strings"

	"git-guider/internal/session"
)

type CommandForm struct {
	Flags       []string
	PathArgs    []int
	DenyFlags   []string
	SubCommands map[string]CommandForm
}

var gitCommandForms = map[string]CommandForm{
	// L1
	"init":   {Flags: []string{"--bare"}, PathArgs: []int{0}},
	"add":    {Flags: []string{"-A", "--all", "-p", "--patch", "-u", "--update", "."}},
	"commit": {Flags: []string{"-m", "--message", "-a", "--all", "--amend", "--allow-empty", "--no-edit"}},
	"status": {Flags: []string{"-s", "--short", "--porcelain", "--branch"}},
	"log":    {Flags: []string{"--oneline", "--graph", "--all", "--format", "-n", "--decorate", "--stat", "--name-only"}},
	"diff":   {Flags: []string{"--cached", "--staged", "--name-only", "--stat", "--shortstat", "HEAD"}},
	"rm":     {Flags: []string{"-f", "--force", "-r", "--cached"}},
	"mv":     {Flags: []string{"-f"}},
	"show":   {Flags: []string{"--stat", "--name-only", "--format"}},

	// L2
	"branch":   {Flags: []string{"-d", "-D", "--list", "-a", "-r", "-v", "-m", "-M"}},
	"checkout": {Flags: []string{"-b", "-B", "--"}},
	"switch":   {Flags: []string{"-c", "-C", "--detach"}},
	"merge":    {Flags: []string{"--no-ff", "--ff-only", "--abort", "--continue", "--squash", "-m"}},
	"tag":      {Flags: []string{"-a", "-m", "-d", "--list", "-l"}},

	// L3
	"clone": {
		Flags:     []string{"--bare", "-b", "--branch", "--depth", "--single-branch"},
		PathArgs:  []int{0, 1},
		DenyFlags: []string{"--template"},
	},
	"remote": {SubCommands: map[string]CommandForm{
		"add":     {PathArgs: []int{1}},
		"remove":  {},
		"rename":  {},
		"get-url": {},
		"show":    {},
	}},
	"push":  {Flags: []string{"-u", "--set-upstream", "--force", "--delete", "--tags"}},
	"pull":  {Flags: []string{"--rebase", "--ff-only", "--no-rebase"}},
	"fetch": {Flags: []string{"--all", "--prune", "--tags"}},

	// L4
	"reset":   {Flags: []string{"--soft", "--mixed", "--hard", "HEAD", "HEAD~1"}},
	"restore": {Flags: []string{"--staged", "--source", "--worktree"}},
	"stash": {SubCommands: map[string]CommandForm{
		"push":  {Flags: []string{"-m", "--message"}},
		"pop":   {},
		"apply": {},
		"drop":  {},
		"list":  {},
		"show":  {},
	}},
	"worktree": {SubCommands: map[string]CommandForm{
		"add":    {PathArgs: []int{0}},
		"list":   {},
		"remove": {PathArgs: []int{0}},
	}},
	"reflog": {Flags: []string{"show", "--all"}},

	// L5
	"rebase":      {Flags: []string{"--onto", "--abort", "--continue", "--skip"}},
	"cherry-pick": {Flags: []string{"--abort", "--continue", "--skip", "-n", "--no-commit"}},
	"rev-parse":   {Flags: []string{"--abbrev-ref", "--short", "--verify", "HEAD"}},
	"cat-file":    {Flags: []string{"-t", "-p", "-s"}},
}

// git config: special handling
var gitConfigWritableKeys = map[string]bool{
	"user.name":            true,
	"user.email":           true,
	"init.defaultbranch":   true,
	"core.editor":          true,
}

var shellCommands = map[string]bool{
	"ls": true, "cat": true, "echo": true, "mkdir": true,
	"rm": true, "pwd": true, "tree": true, "touch": true,
	"mv": true, "cp": true, "clear": true, "cd": true,
	"head": true, "tail": true, "wc": true,
}

// Allowed -c keys for git global flag
var allowedGitConfigKeys = map[string]bool{
	"user.name":          true,
	"user.email":         true,
	"init.defaultbranch": true,
	"core.editor":        true,
}

// ValidateCommand validates a parsed command against the algebra.
func ValidateCommand(cmd *ParsedCommand, sandboxRoot, cwd string) error {
	prog := cmd.Program

	// Check redirect target containment
	if cmd.Redirect != nil {
		if _, err := session.ResolvePath(sandboxRoot, cwd, cmd.Redirect.Target); err != nil {
			return fmt.Errorf("redirect target: %w", err)
		}
	}

	if prog == "git" {
		return validateGitCommand(cmd.Args, sandboxRoot, cwd)
	}
	if prog == "cd" {
		if len(cmd.Args) == 0 {
			return nil
		}
		_, err := session.ResolvePath(sandboxRoot, cwd, cmd.Args[0])
		return err
	}
	if shellCommands[prog] {
		return validateShellPaths(cmd, sandboxRoot, cwd)
	}

	return fmt.Errorf("command not allowed: %s", prog)
}

func validateGitCommand(args []string, sandboxRoot, cwd string) error {
	if len(args) == 0 {
		return fmt.Errorf("git: missing subcommand")
	}

	// Parse and validate global flags first
	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "-C" {
			if i+1 >= len(args) {
				return fmt.Errorf("git -C: missing argument")
			}
			if _, err := session.ResolvePath(sandboxRoot, cwd, args[i+1]); err != nil {
				return fmt.Errorf("git -C: %w", err)
			}
			i += 2
			continue
		}
		if strings.HasPrefix(arg, "--git-dir=") {
			path := strings.TrimPrefix(arg, "--git-dir=")
			if _, err := session.ResolvePath(sandboxRoot, cwd, path); err != nil {
				return fmt.Errorf("git --git-dir: %w", err)
			}
			i++
			continue
		}
		if strings.HasPrefix(arg, "--work-tree=") {
			path := strings.TrimPrefix(arg, "--work-tree=")
			if _, err := session.ResolvePath(sandboxRoot, cwd, path); err != nil {
				return fmt.Errorf("git --work-tree: %w", err)
			}
			i++
			continue
		}
		if strings.HasPrefix(arg, "--exec-path") {
			return fmt.Errorf("git --exec-path: not allowed")
		}
		if arg == "-c" {
			if i+1 >= len(args) {
				return fmt.Errorf("git -c: missing argument")
			}
			kv := args[i+1]
			parts := strings.SplitN(kv, "=", 2)
			key := strings.ToLower(parts[0])
			if !allowedGitConfigKeys[key] {
				return fmt.Errorf("git -c %s: config key not allowed", parts[0])
			}
			i += 2
			continue
		}
		// Not a global flag — this is the subcommand
		break
	}

	if i >= len(args) {
		return fmt.Errorf("git: missing subcommand")
	}

	subcmd := args[i]
	subArgs := args[i+1:]

	// Special handling for git config
	if subcmd == "config" {
		return validateGitConfig(subArgs)
	}

	form, ok := gitCommandForms[subcmd]
	if !ok {
		return fmt.Errorf("git %s: subcommand not allowed", subcmd)
	}

	// If form has SubCommands, validate the second-level subcommand
	if form.SubCommands != nil {
		if len(subArgs) == 0 {
			return fmt.Errorf("git %s: missing subcommand", subcmd)
		}
		sub2 := subArgs[0]
		sub2Form, ok := form.SubCommands[sub2]
		if !ok {
			return fmt.Errorf("git %s %s: subcommand not allowed", subcmd, sub2)
		}
		return validateSubArgs(fmt.Sprintf("git %s %s", subcmd, sub2), sub2Form, subArgs[1:], sandboxRoot, cwd)
	}

	return validateSubArgs("git "+subcmd, form, subArgs, sandboxRoot, cwd)
}

func validateSubArgs(cmdName string, form CommandForm, args []string, sandboxRoot, cwd string) error {
	// Check deny flags
	for _, arg := range args {
		for _, deny := range form.DenyFlags {
			if arg == deny || strings.HasPrefix(arg, deny+"=") {
				return fmt.Errorf("%s: flag %s not allowed", cmdName, deny)
			}
		}
	}

	// Validate flags: extract positional args and check flags
	positional := []string{}
	for j := 0; j < len(args); j++ {
		arg := args[j]
		if strings.HasPrefix(arg, "-") {
			flag := arg
			// Handle --flag=value form
			if idx := strings.Index(arg, "="); idx >= 0 {
				flag = arg[:idx]
			}
			if !isAllowedFlag(flag, form.Flags) {
				return fmt.Errorf("%s: flag %s not allowed", cmdName, arg)
			}
			// Flags that take a value argument
			if flagTakesValue(flag) && !strings.Contains(arg, "=") {
				j++ // skip value
			}
		} else {
			positional = append(positional, arg)
		}
	}

	// Validate path args for containment + transport
	for _, idx := range form.PathArgs {
		if idx < len(positional) {
			path := positional[idx]
			if isTransportURL(path) {
				return fmt.Errorf("%s: remote transport not allowed: %s", cmdName, path)
			}
			if _, err := session.ResolvePath(sandboxRoot, cwd, path); err != nil {
				return fmt.Errorf("%s: path arg: %w", cmdName, err)
			}
		}
	}

	return nil
}

func validateGitConfig(args []string) error {
	// Reject --global, --system, --file
	for _, arg := range args {
		switch arg {
		case "--global", "--system":
			return fmt.Errorf("git config %s: not allowed", arg)
		}
		if strings.HasPrefix(arg, "--file") {
			return fmt.Errorf("git config --file: not allowed")
		}
	}

	// Determine if this is a read or write
	// Read: git config --get <key> or git config <key> (1 positional)
	// Write: git config <key> <value> (2 positionals)
	// List: git config --list / -l
	positional := []string{}
	isGet := false
	for _, arg := range args {
		switch arg {
		case "--get", "--get-all":
			isGet = true
		case "--list", "-l", "--local":
			// allowed
		default:
			if !strings.HasPrefix(arg, "-") {
				positional = append(positional, arg)
			}
		}
	}

	if isGet || len(positional) <= 1 {
		return nil // read is always fine
	}

	// Write: check key
	if len(positional) >= 2 {
		key := strings.ToLower(positional[0])
		if !gitConfigWritableKeys[key] {
			return fmt.Errorf("git config: writing key %q not allowed", positional[0])
		}
	}
	return nil
}

func isAllowedFlag(flag string, allowed []string) bool {
	for _, a := range allowed {
		if flag == a {
			return true
		}
	}
	return false
}

var valueFlagSet = map[string]bool{
	"-m": true, "--message": true, "-n": true, "--format": true,
	"-b": true, "--branch": true, "--depth": true, "--source": true,
	"--onto": true,
}

func flagTakesValue(flag string) bool {
	return valueFlagSet[flag]
}

func isTransportURL(s string) bool {
	prefixes := []string{"https://", "http://", "ssh://", "git://", "file://"}
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	// user@host:path pattern
	if strings.Contains(s, "@") && strings.Contains(s, ":") {
		at := strings.Index(s, "@")
		colon := strings.Index(s, ":")
		if at < colon && !strings.HasPrefix(s[colon:], "://") {
			return true
		}
	}
	return false
}

func validateShellPaths(cmd *ParsedCommand, sandboxRoot, cwd string) error {
	switch cmd.Program {
	case "cat", "head", "tail":
		for _, arg := range cmd.Args {
			if strings.HasPrefix(arg, "-") {
				continue
			}
			if _, err := session.ResolvePath(sandboxRoot, cwd, arg); err != nil {
				return fmt.Errorf("%s: %w", cmd.Program, err)
			}
		}
	case "touch":
		for _, arg := range cmd.Args {
			if strings.HasPrefix(arg, "-") {
				continue
			}
			if _, err := session.ResolvePath(sandboxRoot, cwd, arg); err != nil {
				return fmt.Errorf("touch: %w", err)
			}
		}
	case "mkdir":
		for _, arg := range cmd.Args {
			if strings.HasPrefix(arg, "-") {
				continue
			}
			if _, err := session.ResolvePath(sandboxRoot, cwd, arg); err != nil {
				return fmt.Errorf("mkdir: %w", err)
			}
		}
	case "rm":
		for _, arg := range cmd.Args {
			if strings.HasPrefix(arg, "-") {
				continue
			}
			if _, err := session.ResolvePath(sandboxRoot, cwd, arg); err != nil {
				return fmt.Errorf("rm: %w", err)
			}
		}
	case "cp", "mv":
		for _, arg := range cmd.Args {
			if strings.HasPrefix(arg, "-") {
				continue
			}
			if _, err := session.ResolvePath(sandboxRoot, cwd, arg); err != nil {
				return fmt.Errorf("%s: %w", cmd.Program, err)
			}
		}
	case "ls":
		for _, arg := range cmd.Args {
			if strings.HasPrefix(arg, "-") {
				continue
			}
			if _, err := session.ResolvePath(sandboxRoot, cwd, arg); err != nil {
				return fmt.Errorf("ls: %w", err)
			}
		}
	}
	return nil
}
