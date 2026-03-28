package notes

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

const DefaultRef = "refs/notes/annotate"

func ResolveRef(ns string) string {
	if ns == "" {
		return DefaultRef
	}
	return DefaultRef + "/" + ns
}

// Append adds a JSONL line to the note for the given commit.
// If the commit has no note yet, it creates one.
func Append(ref, commit, line string) error {
	args := []string{"notes", "--ref", ref, "append", "-m", line, commit}
	return run(args...)
}

// Read returns all lines from the note attached to the given commit.
// Returns nil, nil if no note exists.
func Read(ref, commit string) ([]string, error) {
	out, err := output("notes", "--ref", ref, "show", commit)
	if err != nil {
		if strings.Contains(err.Error(), "No note found") {
			return nil, nil
		}
		return nil, err
	}
	raw := strings.TrimSpace(out)
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// ListAnnotated returns a list of commit SHAs that have notes under the given ref.
func ListAnnotated(ref string) ([]string, error) {
	out, err := output("notes", "--ref", ref, "list")
	if err != nil {
		if strings.Contains(err.Error(), "Unexpected end of command stream") ||
			strings.Contains(err.Error(), "exit status") {
			return nil, nil
		}
		return nil, err
	}
	raw := strings.TrimSpace(out)
	if raw == "" {
		return nil, nil
	}
	var commits []string
	for _, line := range strings.Split(raw, "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			commits = append(commits, parts[1])
		}
	}
	return commits, nil
}

// ListRefs returns all annotate namespace refs.
func ListRefs() ([]string, error) {
	out, err := output("for-each-ref", "--format=%(refname)", "refs/notes/annotate")
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(out)
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// Push pushes notes to the remote.
func Push(remote, ref string) error {
	return run("push", remote, ref)
}

// Pull fetches notes from the remote.
func Pull(remote, ref string) error {
	return run("fetch", remote, ref+":"+ref)
}

// PushAll pushes all annotate notes to the remote.
func PushAll(remote string) error {
	return run("push", remote, "refs/notes/annotate/*")
}

// PullAll fetches all annotate notes from the remote.
func PullAll(remote string) error {
	return run("fetch", remote, "refs/notes/annotate/*:refs/notes/annotate/*")
}

// Setup configures the repo to auto-fetch annotate notes and use cat_sort_uniq merge strategy.
func Setup(remote string) error {
	if err := run("config", "--add", "remote."+remote+".fetch", "+refs/notes/annotate/*:refs/notes/annotate/*"); err != nil {
		return fmt.Errorf("failed to configure fetch refspec: %w", err)
	}
	if err := run("config", "notes.mergeStrategy", "cat_sort_uniq"); err != nil {
		return fmt.Errorf("failed to configure merge strategy: %w", err)
	}
	return nil
}

// CommitSubject returns the one-line subject for a commit.
func CommitSubject(commit string) (string, error) {
	return output("log", "-1", "--format=%s", commit)
}

// CommitDate returns the author date for a commit in RFC3339 format.
func CommitDate(commit string) (string, error) {
	return output("log", "-1", "--format=%aI", commit)
}

// RevList returns commit SHAs in the given range.
func RevList(revRange string) ([]string, error) {
	out, err := output("rev-list", revRange)
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(out)
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// ResolveCommit resolves a commit-ish to a full SHA.
func ResolveCommit(commitish string) (string, error) {
	out, err := output("rev-parse", commitish)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// DefaultAuthor returns the git user.name config value.
func DefaultAuthor() string {
	out, _ := output("config", "user.name")
	return strings.TrimSpace(out)
}

func run(args ...string) error {
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(stderr.String()))
	}
	return nil
}

func output(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", args[0], errMsg)
	}
	return strings.TrimSpace(stdout.String()), nil
}
