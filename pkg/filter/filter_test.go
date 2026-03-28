package filter

import (
	"testing"

	"github.com/kennyg/gh-annotate/pkg/annotation"
)

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want bool
	}{
		{"zero value", Options{}, true},
		{"author set", Options{Author: "alice"}, false},
		{"role set", Options{Role: "human"}, false},
		{"tags set", Options{Tags: []string{"bug"}}, false},
		{"thread set", Options{Thread: "t1"}, false},
		{"since set", Options{Since: "2025-01-01"}, false},
		{"until set", Options{Until: "2025-12-31"}, false},
		{"query set", Options{Query: "fix"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opts.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func makeAnnotation(author, role, msg, thread, ts string, tags []string) annotation.Annotation {
	return annotation.Annotation{
		Version: 1,
		Time:    ts,
		Author:  author,
		Role:    annotation.Role(role),
		Tags:    tags,
		Msg:     msg,
		Thread:  thread,
	}
}

func TestMatch(t *testing.T) {
	base := makeAnnotation("Alice", "human", "Fixed the login bug", "thread-1",
		"2025-06-15T10:00:00Z", []string{"bug", "auth"})

	tests := []struct {
		name string
		opts Options
		ann  annotation.Annotation
		want bool
	}{
		// Author filter (case-insensitive)
		{"author exact", Options{Author: "Alice"}, base, true},
		{"author lowercase", Options{Author: "alice"}, base, true},
		{"author uppercase", Options{Author: "ALICE"}, base, true},
		{"author mismatch", Options{Author: "Bob"}, base, false},

		// Role filter
		{"role match", Options{Role: "human"}, base, true},
		{"role mismatch", Options{Role: "agent"}, base, false},

		// Tags filter (all required)
		{"tags single match", Options{Tags: []string{"bug"}}, base, true},
		{"tags all match", Options{Tags: []string{"bug", "auth"}}, base, true},
		{"tags partial mismatch", Options{Tags: []string{"bug", "perf"}}, base, false},
		{"tags none match", Options{Tags: []string{"perf"}}, base, false},
		{"tags filter on annotation with no tags",
			Options{Tags: []string{"bug"}},
			makeAnnotation("Alice", "human", "msg", "", "2025-06-15T10:00:00Z", nil),
			false},

		// Thread filter
		{"thread match", Options{Thread: "thread-1"}, base, true},
		{"thread mismatch", Options{Thread: "thread-2"}, base, false},

		// Query filter (case-insensitive substring)
		{"query match lowercase", Options{Query: "login"}, base, true},
		{"query match uppercase", Options{Query: "LOGIN"}, base, true},
		{"query match mixed case", Options{Query: "Login Bug"}, base, true},
		{"query mismatch", Options{Query: "deploy"}, base, false},

		// Since filter - RFC3339
		{"since before annotation", Options{Since: "2025-06-01T00:00:00Z"}, base, true},
		{"since after annotation", Options{Since: "2025-07-01T00:00:00Z"}, base, false},
		{"since exact time", Options{Since: "2025-06-15T10:00:00Z"}, base, true},

		// Since filter - date-only
		{"since date-only before", Options{Since: "2025-06-01"}, base, true},
		{"since date-only after", Options{Since: "2025-07-01"}, base, false},
		{"since date-only same day", Options{Since: "2025-06-15"}, base, true},

		// Until filter - RFC3339
		{"until after annotation", Options{Until: "2025-07-01T00:00:00Z"}, base, true},
		{"until before annotation", Options{Until: "2025-06-01T00:00:00Z"}, base, false},
		{"until exact time", Options{Until: "2025-06-15T10:00:00Z"}, base, true},

		// Until filter - date-only
		{"until date-only after", Options{Until: "2025-07-01"}, base, true},
		{"until date-only before", Options{Until: "2025-06-01"}, base, false},

		// Empty options matches everything
		{"empty options", Options{}, base, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opts.Match(tt.ann); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchMultipleFilters(t *testing.T) {
	ann := makeAnnotation("Alice", "human", "Fixed login bug", "thread-1",
		"2025-06-15T10:00:00Z", []string{"bug", "auth"})

	tests := []struct {
		name string
		opts Options
		want bool
	}{
		{
			"all filters match",
			Options{
				Author: "alice",
				Role:   "human",
				Tags:   []string{"bug"},
				Thread: "thread-1",
				Since:  "2025-01-01",
				Until:  "2025-12-31",
				Query:  "login",
			},
			true,
		},
		{
			"author matches but role does not",
			Options{Author: "alice", Role: "agent"},
			false,
		},
		{
			"role matches but query does not",
			Options{Role: "human", Query: "deploy"},
			false,
		},
		{
			"tags match but thread does not",
			Options{Tags: []string{"bug"}, Thread: "thread-99"},
			false,
		},
		{
			"time range excludes annotation",
			Options{Author: "alice", Since: "2025-07-01"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opts.Match(ann); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApply(t *testing.T) {
	annotations := []annotation.Annotation{
		makeAnnotation("Alice", "human", "First note", "", "2025-06-01T10:00:00Z", []string{"doc"}),
		makeAnnotation("Bob", "agent", "Second note", "", "2025-06-15T10:00:00Z", []string{"bug"}),
		makeAnnotation("Alice", "agent", "Third note", "", "2025-07-01T10:00:00Z", []string{"bug", "doc"}),
	}

	tests := []struct {
		name      string
		opts      *Options
		wantCount int
	}{
		{"nil opts returns all", nil, 3},
		{"empty opts returns all", &Options{}, 3},
		{"filter by author", &Options{Author: "alice"}, 2},
		{"filter by role", &Options{Role: "agent"}, 2},
		{"filter by tag", &Options{Tags: []string{"bug"}}, 2},
		{"filter by multiple tags", &Options{Tags: []string{"bug", "doc"}}, 1},
		{"no matches returns empty", &Options{Author: "charlie"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Apply(annotations, tt.opts)
			if len(result) != tt.wantCount {
				t.Errorf("Apply() returned %d annotations, want %d", len(result), tt.wantCount)
			}
		})
	}

	t.Run("no matches returns nil slice", func(t *testing.T) {
		result := Apply(annotations, &Options{Author: "charlie"})
		if result != nil {
			t.Errorf("Apply() with no matches should return nil, got %v", result)
		}
	})

	t.Run("nil opts returns original slice", func(t *testing.T) {
		result := Apply(annotations, nil)
		if len(result) != len(annotations) {
			t.Fatalf("expected %d, got %d", len(annotations), len(result))
		}
		// Should be the same slice, not a copy
		if &result[0] != &annotations[0] {
			t.Error("Apply(nil) should return the original slice")
		}
	})
}

func TestSinceUntilEdgeCases(t *testing.T) {
	// Annotation at midnight boundary
	midnight := makeAnnotation("Alice", "human", "midnight note", "",
		"2025-06-15T00:00:00Z", nil)

	tests := []struct {
		name string
		opts Options
		want bool
	}{
		{
			"date-only since matches midnight annotation on same day",
			Options{Since: "2025-06-15"},
			true,
		},
		{
			"date-only until matches midnight annotation on same day",
			Options{Until: "2025-06-15"},
			true,
		},
		{
			"date-only since day after excludes midnight annotation",
			Options{Since: "2025-06-16"},
			false,
		},
		{
			"date-only until day before excludes midnight annotation",
			Options{Until: "2025-06-14"},
			false,
		},
		{
			"rfc3339 since at exact boundary is inclusive",
			Options{Since: "2025-06-15T00:00:00Z"},
			true,
		},
		{
			"rfc3339 until at exact boundary is inclusive",
			Options{Until: "2025-06-15T00:00:00Z"},
			true,
		},
		{
			"rfc3339 since one second after excludes",
			Options{Since: "2025-06-15T00:00:01Z"},
			false,
		},
		{
			"rfc3339 until one second before excludes",
			Options{Until: "2025-06-14T23:59:59Z"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opts.Match(midnight); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}
