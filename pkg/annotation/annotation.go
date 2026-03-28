package annotation

import (
	"encoding/json"
	"fmt"
	"time"
)

type Role string

const (
	RoleHuman Role = "human"
	RoleAgent Role = "agent"
	RoleCI    Role = "ci"
)

func ParseRole(s string) (Role, error) {
	switch Role(s) {
	case RoleHuman, RoleAgent, RoleCI:
		return Role(s), nil
	default:
		return "", fmt.Errorf("invalid role %q: must be one of human, agent, ci", s)
	}
}

type FileRef struct {
	File  string `json:"file"`
	Lines string `json:"lines,omitempty"`
}

type Annotation struct {
	Version int      `json:"v"`
	Time    string   `json:"ts"`
	Author  string   `json:"author"`
	Role    Role     `json:"role"`
	Tags    []string `json:"tags,omitempty"`
	Msg     string   `json:"msg"`
	Thread  string   `json:"thread,omitempty"`
	Ref     *FileRef `json:"ref,omitempty"`
}

// CommitAnnotation wraps an Annotation with the commit it belongs to.
// Used in output for log/search where the consumer needs to know the commit.
type CommitAnnotation struct {
	Commit string `json:"commit"`
	Annotation
}

func New(author string, role Role, msg string) Annotation {
	return Annotation{
		Version: 1,
		Time:    time.Now().UTC().Format(time.RFC3339),
		Author:  author,
		Role:    role,
		Msg:     msg,
	}
}

func (a *Annotation) Validate() error {
	if a.Version == 0 {
		a.Version = 1
	}
	if a.Version != 1 {
		return fmt.Errorf("unsupported schema version: %d", a.Version)
	}
	if a.Author == "" {
		return fmt.Errorf("author is required")
	}
	if _, err := ParseRole(string(a.Role)); err != nil {
		return err
	}
	if a.Msg == "" {
		return fmt.Errorf("msg is required")
	}
	if a.Time == "" {
		a.Time = time.Now().UTC().Format(time.RFC3339)
	}
	return nil
}

func Marshal(a Annotation) (string, error) {
	b, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func Unmarshal(line string) (Annotation, error) {
	var a Annotation
	if err := json.Unmarshal([]byte(line), &a); err != nil {
		return a, fmt.Errorf("invalid annotation JSON: %w", err)
	}
	return a, nil
}

func MarshalCommit(commit string, a Annotation) (string, error) {
	ca := CommitAnnotation{Commit: commit, Annotation: a}
	b, err := json.Marshal(ca)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
