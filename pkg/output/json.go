package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/kennyg/gh-annotate/pkg/annotation"
)

// PrintJSONAnnotation prints a single annotation as a JSONL line.
func PrintJSONAnnotation(w io.Writer, a annotation.Annotation) error {
	b, err := json.Marshal(a)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

// ApplyJQ filters JSONL lines through a jq expression.
func ApplyJQ(lines []string, expr string) ([]string, error) {
	input := strings.Join(lines, "\n")
	cmd := exec.Command("jq", "-c", expr)
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("jq error: %w", err)
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}
