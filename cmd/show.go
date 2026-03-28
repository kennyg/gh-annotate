package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/kennyg/gh-annotate/pkg/annotation"
	"github.com/kennyg/gh-annotate/pkg/filter"
	"github.com/kennyg/gh-annotate/pkg/notes"
	"github.com/kennyg/gh-annotate/pkg/output"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "show [<commit>]",
		Short: "Show annotations for a commit",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runShow,
	}

	cmd.Flags().Bool("json", false, "Output as JSONL")
	cmd.Flags().StringP("jq", "q", "", "Filter JSON output with jq expression")
	cmd.Flags().String("ns", "", "Namespace suffix")
	cmd.Flags().Bool("all-ns", false, "Show annotations from all namespaces")
	cmd.Flags().StringP("author", "a", "", "Filter by author")
	cmd.Flags().StringP("role", "r", "", "Filter by role")
	cmd.Flags().StringP("tags", "t", "", "Filter: annotation must have all specified tags (comma-separated)")

	rootCmd.AddCommand(cmd)
}

func runShow(cmd *cobra.Command, args []string) error {
	commit := "HEAD"
	if len(args) > 0 {
		commit = args[0]
	}

	resolved, err := notes.ResolveCommit(commit)
	if err != nil {
		return fmt.Errorf("cannot resolve commit %q: %w", commit, err)
	}

	refs, err := collectRefs(cmd)
	if err != nil {
		return err
	}

	opts := buildFilterOpts(cmd)
	jsonMode, _ := cmd.Flags().GetBool("json")
	jqExpr, _ := cmd.Flags().GetString("jq")
	tty := output.IsTTY() && !jsonMode

	annotations := readAnnotations(refs, resolved)

	annotations = filter.Apply(annotations, opts)

	if len(annotations) == 0 {
		fmt.Fprintln(os.Stderr, "No annotations found")
		os.Exit(4)
	}

	if jsonMode || jqExpr != "" {
		return printJSON(os.Stdout, annotations, jqExpr)
	}

	subject, _ := notes.CommitSubject(resolved)
	date, _ := notes.CommitDate(resolved)
	output.PrintCommitHeader(os.Stdout, resolved, subject, date, tty)
	fmt.Println()
	for _, a := range annotations {
		output.PrintAnnotation(os.Stdout, a, tty)
		fmt.Println()
	}
	return nil
}

func printJSON(w *os.File, annotations []annotation.Annotation, jqExpr string) error {
	if jqExpr == "" {
		for _, a := range annotations {
			if err := output.PrintJSONAnnotation(w, a); err != nil {
				return err
			}
		}
		return nil
	}

	var lines []string
	for _, a := range annotations {
		line, err := annotation.Marshal(a)
		if err != nil {
			return err
		}
		lines = append(lines, line)
	}

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

func collectRefs(cmd *cobra.Command) ([]string, error) {
	allNS, _ := cmd.Flags().GetBool("all-ns")
	if allNS {
		refs, err := notes.ListRefs()
		if err != nil {
			return nil, err
		}
		if len(refs) == 0 {
			return []string{notes.DefaultRef}, nil
		}
		// Include the default ref if it's not already in the list
		found := false
		for _, r := range refs {
			if r == notes.DefaultRef {
				found = true
				break
			}
		}
		if !found {
			refs = append([]string{notes.DefaultRef}, refs...)
		}
		return refs, nil
	}
	ns, _ := cmd.Flags().GetString("ns")
	return []string{notes.ResolveRef(ns)}, nil
}

func buildFilterOpts(cmd *cobra.Command) *filter.Options {
	opts := &filter.Options{}
	opts.Author, _ = cmd.Flags().GetString("author")
	opts.Role, _ = cmd.Flags().GetString("role")
	if tags, _ := cmd.Flags().GetString("tags"); tags != "" {
		opts.Tags = append(opts.Tags, splitTags(tags)...)
	}
	if cmd.Flags().Lookup("thread") != nil {
		opts.Thread, _ = cmd.Flags().GetString("thread")
	}
	if cmd.Flags().Lookup("since") != nil {
		opts.Since, _ = cmd.Flags().GetString("since")
	}
	if cmd.Flags().Lookup("until") != nil {
		opts.Until, _ = cmd.Flags().GetString("until")
	}
	return opts
}

func splitTags(s string) []string {
	var tags []string
	for _, t := range strings.Split(s, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

func readAnnotations(refs []string, commit string) []annotation.Annotation {
	var result []annotation.Annotation
	for _, ref := range refs {
		lines, err := notes.Read(ref, commit)
		if err != nil {
			continue
		}
		for _, line := range lines {
			a, err := annotation.Unmarshal(line)
			if err != nil {
				continue
			}
			result = append(result, a)
		}
	}
	return result
}

func collectAnnotatedCommits(refs []string) []string {
	seen := map[string]bool{}
	var commits []string
	for _, ref := range refs {
		annotated, err := notes.ListAnnotated(ref)
		if err != nil {
			continue
		}
		for _, c := range annotated {
			if !seen[c] {
				seen[c] = true
				commits = append(commits, c)
			}
		}
	}
	return commits
}
