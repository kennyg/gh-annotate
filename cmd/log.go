package cmd

import (
	"fmt"
	"os"

	"github.com/kennyg/gh-annotate/pkg/annotation"
	"github.com/kennyg/gh-annotate/pkg/filter"
	"github.com/kennyg/gh-annotate/pkg/notes"
	"github.com/kennyg/gh-annotate/pkg/output"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "log [<revision-range>]",
		Short: "Show annotations across a commit range",
		Long: `Show annotations for commits in a revision range.

If no range is specified, shows all annotated commits on the current branch.
Accepts any git revision range syntax (e.g., main..HEAD, HEAD~10..).`,
		Args: cobra.MaximumNArgs(1),
		RunE: runLog,
	}

	cmd.Flags().Bool("json", false, "Output as JSONL")
	cmd.Flags().StringP("jq", "q", "", "Filter JSON output with jq expression")
	cmd.Flags().String("ns", "", "Namespace suffix")
	cmd.Flags().Bool("all-ns", false, "Show annotations from all namespaces")
	cmd.Flags().StringP("author", "a", "", "Filter by author")
	cmd.Flags().StringP("role", "r", "", "Filter by role")
	cmd.Flags().StringP("tags", "t", "", "Filter by tags (comma-separated)")
	cmd.Flags().String("thread", "", "Filter by thread")
	cmd.Flags().IntP("limit", "L", 30, "Max commits to show")
	cmd.Flags().String("since", "", "Only annotations after this date")
	cmd.Flags().String("until", "", "Only annotations before this date")

	rootCmd.AddCommand(cmd)
}

func runLog(cmd *cobra.Command, args []string) error {
	refs, err := collectRefs(cmd)
	if err != nil {
		return err
	}

	opts := buildFilterOpts(cmd)
	jsonMode, _ := cmd.Flags().GetBool("json")
	jqExpr, _ := cmd.Flags().GetString("jq")
	limit, _ := cmd.Flags().GetInt("limit")
	tty := output.IsTTY() && !jsonMode

	// Determine which commits to check
	var commits []string
	if len(args) > 0 {
		commits, err = notes.RevList(args[0])
		if err != nil {
			return fmt.Errorf("invalid revision range: %w", err)
		}
	} else {
		commits = collectAnnotatedCommits(refs)
	}

	found := 0
	var jsonLines []string

	for _, commit := range commits {
		if found >= limit {
			break
		}

		annotations := readAnnotations(refs, commit)

		annotations = filter.Apply(annotations, opts)
		if len(annotations) == 0 {
			continue
		}

		found++

		if jsonMode || jqExpr != "" {
			for _, a := range annotations {
				line, err := annotation.MarshalCommit(commit, a)
				if err != nil {
					return err
				}
				jsonLines = append(jsonLines, line)
			}
			continue
		}

		subject, _ := notes.CommitSubject(commit)
		date, _ := notes.CommitDate(commit)
		output.PrintCommitHeader(os.Stdout, commit, subject, date, tty)
		fmt.Println()
		for _, a := range annotations {
			output.PrintAnnotation(os.Stdout, a, tty)
			fmt.Println()
		}
	}

	if jsonMode || jqExpr != "" {
		return printJSONLines(os.Stdout, jsonLines, jqExpr)
	}

	if found == 0 {
		fmt.Fprintln(os.Stderr, "No annotations found")
		os.Exit(4)
	}

	return nil
}

func printJSONLines(w *os.File, lines []string, jqExpr string) error {
	if jqExpr != "" {
		filtered, err := output.ApplyJQ(lines, jqExpr)
		if err != nil {
			return err
		}
		for _, l := range filtered {
			if _, err := fmt.Fprintln(w, l); err != nil {
				return err
			}
		}
		return nil
	}
	for _, l := range lines {
		if _, err := fmt.Fprintln(w, l); err != nil {
			return err
		}
	}
	return nil
}
