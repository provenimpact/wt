package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion <shell>",
	Short: "Output shell completion script",
	Long:  "Output a shell completion script for the specified shell.\n\nSupported shells: bash, zsh, fish\n\nUsage:\n  eval \"$(wt completion bash)\"   # for .bashrc\n  eval \"$(wt completion zsh)\"    # for .zshrc\n  wt completion fish | source    # for config.fish",
	Args:  cobra.ExactArgs(1),
	RunE:  runCompletion,
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

func runCompletion(cmd *cobra.Command, args []string) error {
	switch args[0] {
	case "bash":
		return rootCmd.GenBashCompletionV2(os.Stdout, true)
	case "zsh":
		return rootCmd.GenZshCompletion(os.Stdout)
	case "fish":
		return rootCmd.GenFishCompletion(os.Stdout, true)
	default:
		return fmt.Errorf("unsupported shell %q; supported: bash, zsh, fish", args[0])
	}
}
