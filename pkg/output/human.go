package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kennyg/gh-annotate/pkg/annotation"
)

var (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorBlue   = "\033[34m"
)

func IsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func roleBadge(role annotation.Role) string {
	switch role {
	case annotation.RoleAgent:
		return colorCyan + "[agent]" + colorReset
	case annotation.RoleCI:
		return colorBlue + "[ci]" + colorReset
	default:
		return colorGreen + "[human]" + colorReset
	}
}

func roleBadgePlain(role annotation.Role) string {
	return "[" + string(role) + "]"
}

func relativeTime(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("2006-01-02")
	}
}

// PrintAnnotation prints a single annotation in human-readable format.
func PrintAnnotation(w io.Writer, a annotation.Annotation, tty bool) {
	if tty {
		printAnnotationColor(w, a)
	} else {
		printAnnotationPlain(w, a)
	}
}

func printAnnotationColor(w io.Writer, a annotation.Annotation) {
	tags := ""
	if len(a.Tags) > 0 {
		tags = "  " + colorDim + strings.Join(a.Tags, ", ") + colorReset
	}
	fmt.Fprintf(w, "  %s %s%s%s  %s%s%s\n",
		roleBadge(a.Role),
		colorBold, a.Author, colorReset,
		colorDim, relativeTime(a.Time), colorReset)
	if tags != "" {
		fmt.Fprintf(w, "  %s\n", tags)
	}
	for _, line := range strings.Split(a.Msg, "\n") {
		fmt.Fprintf(w, "  %s\n", line)
	}
	if a.Ref != nil {
		loc := a.Ref.File
		if a.Ref.Lines != "" {
			loc += ":" + a.Ref.Lines
		}
		fmt.Fprintf(w, "  %s> %s%s\n", colorDim, loc, colorReset)
	}
}

func printAnnotationPlain(w io.Writer, a annotation.Annotation) {
	tags := ""
	if len(a.Tags) > 0 {
		tags = "  " + strings.Join(a.Tags, ", ")
	}
	fmt.Fprintf(w, "  %s %s  %s\n", roleBadgePlain(a.Role), a.Author, a.Time)
	if tags != "" {
		fmt.Fprintf(w, "  %s\n", tags)
	}
	for _, line := range strings.Split(a.Msg, "\n") {
		fmt.Fprintf(w, "  %s\n", line)
	}
	if a.Ref != nil {
		loc := a.Ref.File
		if a.Ref.Lines != "" {
			loc += ":" + a.Ref.Lines
		}
		fmt.Fprintf(w, "  > %s\n", loc)
	}
}

// PrintCommitHeader prints a commit header line.
func PrintCommitHeader(w io.Writer, sha, subject, date string, tty bool) {
	short := sha
	if len(sha) > 7 {
		short = sha[:7]
	}
	if tty {
		fmt.Fprintf(w, "%s%s%s %s %s(%s)%s\n",
			colorYellow, short, colorReset,
			subject,
			colorDim, date, colorReset)
	} else {
		fmt.Fprintf(w, "%s %s (%s)\n", short, subject, date)
	}
}

// PrintLsEntry prints a single line for the ls command.
func PrintLsEntry(w io.Writer, sha, subject string, total int, roleCounts map[annotation.Role]int, tty bool) {
	short := sha
	if len(sha) > 7 {
		short = sha[:7]
	}
	noun := "annotations"
	if total == 1 {
		noun = "annotation"
	}
	parts := []string{}
	for _, r := range []annotation.Role{annotation.RoleAgent, annotation.RoleHuman, annotation.RoleCI} {
		if c, ok := roleCounts[r]; ok && c > 0 {
			parts = append(parts, fmt.Sprintf("%s: %d", r, c))
		}
	}
	detail := strings.Join(parts, ", ")
	if tty {
		fmt.Fprintf(w, "%s%s%s %-50s %s%d %s%s (%s)\n",
			colorYellow, short, colorReset,
			subject,
			colorDim, total, noun, colorReset,
			detail)
	} else {
		fmt.Fprintf(w, "%s %-50s %d %s (%s)\n", short, subject, total, noun, detail)
	}
}
