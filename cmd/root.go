package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gh-annotate",
	Short: "Annotate git objects with structured notes for agent-human collaboration",
	Long: `gh-annotate uses git notes to create a sideband channel in your repository.
Agents and humans can attach structured annotations (JSONL) to commits
without modifying the commit history.

Annotations are stored in refs/notes/annotate and can be synced with remotes.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}
