package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kennyg/gh-annotate/pkg/annotation"
	"github.com/kennyg/gh-annotate/pkg/notes"
	"github.com/kennyg/gh-annotate/pkg/output"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List annotated commits",
		Args:  cobra.NoArgs,
		RunE:  runLs,
	}

	cmd.Flags().Bool("json", false, "Output as JSON")
	cmd.Flags().String("ns", "", "Namespace suffix")
	cmd.Flags().Bool("all-ns", false, "List from all namespaces")
	cmd.Flags().IntP("limit", "L", 30, "Max results")

	rootCmd.AddCommand(cmd)
}

type lsEntry struct {
	Commit     string         `json:"commit"`
	Subject    string         `json:"subject"`
	Total      int            `json:"total"`
	RoleCounts map[string]int `json:"roles"`
}

func runLs(cmd *cobra.Command, args []string) error {
	refs, err := collectRefs(cmd)
	if err != nil {
		return err
	}

	jsonMode, _ := cmd.Flags().GetBool("json")
	limit, _ := cmd.Flags().GetInt("limit")
	tty := output.IsTTY() && !jsonMode

	commits := collectAnnotatedCommits(refs)

	if len(commits) == 0 {
		fmt.Fprintln(os.Stderr, "No annotated commits found")
		os.Exit(4)
	}

	count := 0
	for _, commit := range commits {
		if count >= limit {
			break
		}

		annotations := readAnnotations(refs, commit)

		if len(annotations) == 0 {
			continue
		}

		roleCounts := map[annotation.Role]int{}
		for _, a := range annotations {
			roleCounts[a.Role]++
		}

		subject, _ := notes.CommitSubject(commit)

		if jsonMode {
			rc := map[string]int{}
			for r, c := range roleCounts {
				rc[string(r)] = c
			}
			entry := lsEntry{
				Commit:     commit,
				Subject:    subject,
				Total:      len(annotations),
				RoleCounts: rc,
			}
			b, _ := json.Marshal(entry)
			fmt.Fprintln(os.Stdout, string(b))
		} else {
			output.PrintLsEntry(os.Stdout, commit, subject, len(annotations), roleCounts, tty)
		}

		count++
	}

	return nil
}
