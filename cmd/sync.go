package cmd

import (
	"fmt"
	"os"

	"github.com/kennyg/gh-annotate/pkg/notes"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Push and pull annotations to/from remote",
		Long: `Synchronize annotations with a remote repository.

With no flags, pulls then pushes (safe round-trip).
Use --setup to configure the repo for automatic note fetching.`,
		Args: cobra.NoArgs,
		RunE: runSync,
	}

	cmd.Flags().Bool("push", false, "Push annotations to remote")
	cmd.Flags().Bool("pull", false, "Pull annotations from remote")
	cmd.Flags().String("ns", "", "Specific namespace (default: all under refs/notes/annotate)")
	cmd.Flags().String("remote", "origin", "Remote name")
	cmd.Flags().Bool("setup", false, "Configure repo to auto-fetch annotate notes and set merge strategy")

	rootCmd.AddCommand(cmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	remote, _ := cmd.Flags().GetString("remote")
	setup, _ := cmd.Flags().GetBool("setup")
	pushOnly, _ := cmd.Flags().GetBool("push")
	pullOnly, _ := cmd.Flags().GetBool("pull")
	ns, _ := cmd.Flags().GetString("ns")

	if setup {
		if err := notes.Setup(remote); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Configured auto-fetch for annotate notes and cat_sort_uniq merge strategy")
		return nil
	}

	doPull := pullOnly || (!pushOnly && !pullOnly)
	doPush := pushOnly || (!pushOnly && !pullOnly)

	if ns != "" {
		ref := notes.ResolveRef(ns)
		if doPull {
			fmt.Fprintf(os.Stderr, "Pulling %s from %s...\n", ref, remote)
			if err := notes.Pull(remote, ref); err != nil {
				return fmt.Errorf("pull failed: %w", err)
			}
		}
		if doPush {
			fmt.Fprintf(os.Stderr, "Pushing %s to %s...\n", ref, remote)
			if err := notes.Push(remote, ref); err != nil {
				return fmt.Errorf("push failed: %w", err)
			}
		}
	} else {
		if doPull {
			fmt.Fprintf(os.Stderr, "Pulling annotations from %s...\n", remote)
			if err := notes.PullAll(remote); err != nil {
				// Try pulling just the default ref if glob fails
				if err2 := notes.Pull(remote, notes.DefaultRef); err2 != nil {
					return fmt.Errorf("pull failed: %w", err)
				}
			}
		}
		if doPush {
			fmt.Fprintf(os.Stderr, "Pushing annotations to %s...\n", remote)
			if err := notes.PushAll(remote); err != nil {
				// Try pushing just the default ref if glob fails
				if err2 := notes.Push(remote, notes.DefaultRef); err2 != nil {
					return fmt.Errorf("push failed: %w", err)
				}
			}
		}
	}

	fmt.Fprintln(os.Stderr, "Sync complete")
	return nil
}
