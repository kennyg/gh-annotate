package filter

import (
	"strings"
	"time"

	"github.com/kennyg/gh-annotate/pkg/annotation"
)

type Options struct {
	Author string
	Role   string
	Tags   []string
	Thread string
	Since  string
	Until  string
	Query  string // full-text substring search on msg
}

func (o *Options) IsEmpty() bool {
	return o.Author == "" && o.Role == "" && len(o.Tags) == 0 &&
		o.Thread == "" && o.Since == "" && o.Until == "" && o.Query == ""
}

func (o *Options) Match(a annotation.Annotation) bool {
	if o.Author != "" && !strings.EqualFold(a.Author, o.Author) {
		return false
	}
	if o.Role != "" && string(a.Role) != o.Role {
		return false
	}
	if len(o.Tags) > 0 {
		tagSet := make(map[string]bool, len(a.Tags))
		for _, t := range a.Tags {
			tagSet[t] = true
		}
		for _, required := range o.Tags {
			if !tagSet[required] {
				return false
			}
		}
	}
	if o.Thread != "" && a.Thread != o.Thread {
		return false
	}
	if o.Since != "" {
		if t, err := time.Parse(time.RFC3339, a.Time); err == nil {
			if since, err := parseFlexibleTime(o.Since); err == nil {
				if t.Before(since) {
					return false
				}
			}
		}
	}
	if o.Until != "" {
		if t, err := time.Parse(time.RFC3339, a.Time); err == nil {
			if until, err := parseFlexibleTime(o.Until); err == nil {
				if t.After(until) {
					return false
				}
			}
		}
	}
	if o.Query != "" && !strings.Contains(strings.ToLower(a.Msg), strings.ToLower(o.Query)) {
		return false
	}
	return true
}

func Apply(annotations []annotation.Annotation, opts *Options) []annotation.Annotation {
	if opts == nil || opts.IsEmpty() {
		return annotations
	}
	var result []annotation.Annotation
	for _, a := range annotations {
		if opts.Match(a) {
			result = append(result, a)
		}
	}
	return result
}

// parseFlexibleTime tries RFC3339, then date-only format.
func parseFlexibleTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}
