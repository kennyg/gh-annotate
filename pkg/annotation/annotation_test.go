package annotation

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParseRole(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Role
		wantErr bool
	}{
		{"human", "human", RoleHuman, false},
		{"agent", "agent", RoleAgent, false},
		{"ci", "ci", RoleCI, false},
		{"invalid", "bot", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRole(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseRole(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseRole(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	before := time.Now().UTC()
	a := New("alice", RoleHuman, "test message")
	after := time.Now().UTC()

	if a.Version != 1 {
		t.Errorf("Version = %d, want 1", a.Version)
	}
	if a.Author != "alice" {
		t.Errorf("Author = %q, want %q", a.Author, "alice")
	}
	if a.Role != RoleHuman {
		t.Errorf("Role = %q, want %q", a.Role, RoleHuman)
	}
	if a.Msg != "test message" {
		t.Errorf("Msg = %q, want %q", a.Msg, "test message")
	}
	if a.Tags != nil {
		t.Errorf("Tags = %v, want nil", a.Tags)
	}
	if a.Thread != "" {
		t.Errorf("Thread = %q, want empty", a.Thread)
	}
	if a.Ref != nil {
		t.Errorf("Ref = %v, want nil", a.Ref)
	}

	ts, err := time.Parse(time.RFC3339, a.Time)
	if err != nil {
		t.Fatalf("Time %q is not valid RFC3339: %v", a.Time, err)
	}
	if ts.Before(before.Truncate(time.Second)) || ts.After(after.Add(time.Second)) {
		t.Errorf("Time %v not between %v and %v", ts, before, after)
	}
}

func TestValidate(t *testing.T) {
	validAnnotation := func() Annotation {
		return Annotation{
			Version: 1,
			Time:    "2025-01-01T00:00:00Z",
			Author:  "alice",
			Role:    RoleHuman,
			Msg:     "hello",
		}
	}

	t.Run("valid annotation", func(t *testing.T) {
		a := validAnnotation()
		if err := a.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("missing author", func(t *testing.T) {
		a := validAnnotation()
		a.Author = ""
		err := a.Validate()
		if err == nil {
			t.Fatal("expected error for missing author")
		}
		if !strings.Contains(err.Error(), "author") {
			t.Errorf("error %q should mention author", err)
		}
	})

	t.Run("missing msg", func(t *testing.T) {
		a := validAnnotation()
		a.Msg = ""
		err := a.Validate()
		if err == nil {
			t.Fatal("expected error for missing msg")
		}
		if !strings.Contains(err.Error(), "msg") {
			t.Errorf("error %q should mention msg", err)
		}
	})

	t.Run("invalid role", func(t *testing.T) {
		a := validAnnotation()
		a.Role = "bot"
		err := a.Validate()
		if err == nil {
			t.Fatal("expected error for invalid role")
		}
		if !strings.Contains(err.Error(), "invalid role") {
			t.Errorf("error %q should mention invalid role", err)
		}
	})

	t.Run("version 0 defaults to 1", func(t *testing.T) {
		a := validAnnotation()
		a.Version = 0
		if err := a.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Version != 1 {
			t.Errorf("Version = %d, want 1 after defaulting", a.Version)
		}
	})

	t.Run("unsupported version", func(t *testing.T) {
		a := validAnnotation()
		a.Version = 99
		err := a.Validate()
		if err == nil {
			t.Fatal("expected error for unsupported version")
		}
		if !strings.Contains(err.Error(), "unsupported schema version") {
			t.Errorf("error %q should mention unsupported schema version", err)
		}
	})

	t.Run("empty time gets filled", func(t *testing.T) {
		a := validAnnotation()
		a.Time = ""
		before := time.Now().UTC()
		if err := a.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Time == "" {
			t.Fatal("Time should have been filled in")
		}
		ts, err := time.Parse(time.RFC3339, a.Time)
		if err != nil {
			t.Fatalf("filled Time %q is not valid RFC3339: %v", a.Time, err)
		}
		if ts.Before(before.Truncate(time.Second)) {
			t.Errorf("filled Time %v is before test start %v", ts, before)
		}
	})
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		annotation Annotation
	}{
		{
			name: "basic",
			annotation: Annotation{
				Version: 1,
				Time:    "2025-01-01T00:00:00Z",
				Author:  "alice",
				Role:    RoleHuman,
				Msg:     "basic message",
			},
		},
		{
			name: "with tags",
			annotation: Annotation{
				Version: 1,
				Time:    "2025-01-01T00:00:00Z",
				Author:  "bob",
				Role:    RoleAgent,
				Msg:     "tagged message",
				Tags:    []string{"refactor", "perf"},
			},
		},
		{
			name: "with ref",
			annotation: Annotation{
				Version: 1,
				Time:    "2025-01-01T00:00:00Z",
				Author:  "ci-bot",
				Role:    RoleCI,
				Msg:     "lint failure",
				Ref:     &FileRef{File: "main.go", Lines: "10-20"},
			},
		},
		{
			name: "with thread",
			annotation: Annotation{
				Version: 1,
				Time:    "2025-01-01T00:00:00Z",
				Author:  "alice",
				Role:    RoleHuman,
				Msg:     "follow-up",
				Thread:  "abc123",
			},
		},
		{
			name: "all fields",
			annotation: Annotation{
				Version: 1,
				Time:    "2025-06-15T12:30:00Z",
				Author:  "agent-x",
				Role:    RoleAgent,
				Msg:     "full annotation",
				Tags:    []string{"bug", "fix"},
				Thread:  "thread-42",
				Ref:     &FileRef{File: "pkg/foo.go", Lines: "5"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := Marshal(tt.annotation)
			if err != nil {
				t.Fatalf("Marshal() error: %v", err)
			}

			got, err := Unmarshal(s)
			if err != nil {
				t.Fatalf("Unmarshal() error: %v", err)
			}

			// Compare via JSON to handle nil vs empty slice differences cleanly.
			wantJSON, _ := json.Marshal(tt.annotation)
			gotJSON, _ := json.Marshal(got)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("round-trip mismatch:\n  got:  %s\n  want: %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestMarshalCommit(t *testing.T) {
	a := Annotation{
		Version: 1,
		Time:    "2025-01-01T00:00:00Z",
		Author:  "alice",
		Role:    RoleHuman,
		Msg:     "test commit annotation",
	}

	s, err := MarshalCommit("abc123def", a)
	if err != nil {
		t.Fatalf("MarshalCommit() error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	commit, ok := raw["commit"]
	if !ok {
		t.Fatal("output missing 'commit' field")
	}
	if commit != "abc123def" {
		t.Errorf("commit = %q, want %q", commit, "abc123def")
	}

	// Verify embedded annotation fields are present at top level.
	if raw["author"] != "alice" {
		t.Errorf("author = %v, want %q", raw["author"], "alice")
	}
	if raw["msg"] != "test commit annotation" {
		t.Errorf("msg = %v, want %q", raw["msg"], "test commit annotation")
	}
}

func TestUnmarshalInvalidJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"not json", "hello world"},
		{"incomplete json", `{"author": "alice"`},
		{"array instead of object", `[1, 2, 3]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Unmarshal(tt.input)
			if err == nil {
				t.Fatalf("Unmarshal(%q) expected error, got nil", tt.input)
			}
			if !strings.Contains(err.Error(), "invalid annotation JSON") {
				t.Errorf("error %q should contain 'invalid annotation JSON'", err)
			}
		})
	}
}
