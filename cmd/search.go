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
		Use:   "search <query>",
		Short: "Search annotations by message content",
		Long: `Full-text substring search across all annotation messages.

Use an empty string "" with filters to search by metadata only.`,
		Args: cobra.ExactArgs(1),
		RunE: runSearch,
	}

	cmd.Flags().Bool("json", false, "Output as JSONL")
	cmd.Flags().StringP("jq", "q", "", "Filter JSON output with jq expression")
	cmd.Flags().String("ns", "", "Namespace suffix")
	cmd.Flags().Bool("all-ns", false, "Search all namespaces")
	cmd.Flags().StringP("author", "a", "", "Filter by author")
	cmd.Flags().StringP("role", "r", "", "Filter by role")
	cmd.Flags().StringP("tags", "t", "", "Filter by tags (comma-separated)")
	cmd.Flags().IntP("limit", "L", 30, "Max results")

	rootCmd.AddCommand(cmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	refs, err := collectRefs(cmd)
	if err != nil {
		return err
	}

	opts := buildFilterOpts(cmd)
	opts.Query = query

	jsonMode, _ := cmd.Flags().GetBool("json")
	jqExpr, _ := cmd.Flags().GetString("jq")
	limit, _ := cmd.Flags().GetInt("limit")
	tty := output.IsTTY() && !jsonMode

	commits := collectAnnotatedCommits(refs)

	found := 0
	var jsonLines []string

	for _, commit := range commits {
		if found >= limit {
			break
		}

		matched := readAnnotations(refs, commit)

		matched = filter.Apply(matched, opts)
		if len(matched) == 0 {
			continue
		}

		found++

		if jsonMode || jqExpr != "" {
			for _, a := range matched {
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
		for _, a := range matched {
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
