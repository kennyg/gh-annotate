package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kennyg/gh-annotate/pkg/annotation"
	"github.com/kennyg/gh-annotate/pkg/notes"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "add [<commit>]",
		Short: "Add an annotation to a commit",
		Long: `Add a structured annotation to a commit (defaults to HEAD).

The annotation is stored as a JSONL line in git notes under refs/notes/annotate.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runAdd,
	}

	cmd.Flags().StringP("msg", "m", "", "Annotation message")
	cmd.Flags().StringP("file", "F", "", "Read message from file")
	cmd.Flags().StringP("author", "a", "", "Author name (default: $GH_ANNOTATE_AUTHOR or git user.name)")
	cmd.Flags().StringP("role", "r", "", "Role: human, agent, or ci (default: $GH_ANNOTATE_ROLE or human)")
	cmd.Flags().StringP("tags", "t", "", "Comma-separated tags")
	cmd.Flags().String("thread", "", "Thread identifier")
	cmd.Flags().String("ref-file", "", "File path this annotation refers to")
	cmd.Flags().String("ref-lines", "", "Line range (e.g., 42-58)")
	cmd.Flags().String("ns", "", "Namespace suffix")
	cmd.Flags().Bool("json-input", false, "Read a JSON annotation from stdin")
	cmd.Flags().Bool("batch", false, "Read JSONL annotations from stdin (each line must have a \"commit\" field)")

	rootCmd.AddCommand(cmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	ns, _ := cmd.Flags().GetString("ns")
	ref := notes.ResolveRef(ns)

	jsonInput, _ := cmd.Flags().GetBool("json-input")
	batch, _ := cmd.Flags().GetBool("batch")

	if batch {
		return runAddBatch(ref, os.Stdin)
	}

	commit := "HEAD"
	if len(args) > 0 {
		commit = args[0]
	}
	resolved, err := notes.ResolveCommit(commit)
	if err != nil {
		return fmt.Errorf("cannot resolve commit %q: %w", commit, err)
	}

	var a annotation.Annotation

	if jsonInput {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		if err := json.Unmarshal(data, &a); err != nil {
			return fmt.Errorf("invalid JSON input: %w", err)
		}
	} else {
		a, err = buildAnnotation(cmd)
		if err != nil {
			return err
		}
	}

	if err := a.Validate(); err != nil {
		return err
	}

	line, err := annotation.Marshal(a)
	if err != nil {
		return err
	}

	if err := notes.Append(ref, resolved, line); err != nil {
		return fmt.Errorf("failed to add annotation: %w", err)
	}

	short := resolved
	if len(short) > 7 {
		short = short[:7]
	}
	fmt.Fprintf(os.Stderr, "Annotation added to %s\n", short)
	return nil
}

func runAddBatch(ref string, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var bl struct {
			Commit string `json:"commit"`
			annotation.Annotation
		}
		if err := json.Unmarshal([]byte(line), &bl); err != nil {
			return fmt.Errorf("invalid JSON on line %d: %w", count+1, err)
		}
		if bl.Commit == "" {
			return fmt.Errorf("line %d: missing \"commit\" field", count+1)
		}
		commit := bl.Commit
		a := bl.Annotation

		if err := a.Validate(); err != nil {
			return fmt.Errorf("line %d: %w", count+1, err)
		}

		resolved, err := notes.ResolveCommit(commit)
		if err != nil {
			return fmt.Errorf("line %d: cannot resolve commit %q: %w", count+1, commit, err)
		}

		noteLine, err := annotation.Marshal(a)
		if err != nil {
			return err
		}

		if err := notes.Append(ref, resolved, noteLine); err != nil {
			return fmt.Errorf("line %d: failed to add annotation: %w", count+1, err)
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Added %d annotations\n", count)
	return nil
}

func buildAnnotation(cmd *cobra.Command) (annotation.Annotation, error) {
	msg, _ := cmd.Flags().GetString("msg")
	filePath, _ := cmd.Flags().GetString("file")

	if msg == "" && filePath == "" {
		return annotation.Annotation{}, fmt.Errorf("--msg or --file is required")
	}

	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return annotation.Annotation{}, fmt.Errorf("reading file: %w", err)
		}
		msg = strings.TrimSpace(string(data))
	}

	author := resolveAuthor(cmd)
	role := resolveRole(cmd)

	r, err := annotation.ParseRole(role)
	if err != nil {
		return annotation.Annotation{}, err
	}

	a := annotation.New(author, r, msg)

	if tags, _ := cmd.Flags().GetString("tags"); tags != "" {
		for _, t := range strings.Split(tags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				a.Tags = append(a.Tags, t)
			}
		}
	}

	if thread, _ := cmd.Flags().GetString("thread"); thread != "" {
		a.Thread = thread
	}

	refFile, _ := cmd.Flags().GetString("ref-file")
	refLines, _ := cmd.Flags().GetString("ref-lines")
	if refFile != "" {
		a.Ref = &annotation.FileRef{File: refFile, Lines: refLines}
	}

	return a, nil
}

func resolveAuthor(cmd *cobra.Command) string {
	if a, _ := cmd.Flags().GetString("author"); a != "" {
		return a
	}
	if a := os.Getenv("GH_ANNOTATE_AUTHOR"); a != "" {
		return a
	}
	return notes.DefaultAuthor()
}

func resolveRole(cmd *cobra.Command) string {
	if r, _ := cmd.Flags().GetString("role"); r != "" {
		return r
	}
	if r := os.Getenv("GH_ANNOTATE_ROLE"); r != "" {
		return r
	}
	return "human"
}
