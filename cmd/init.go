package cmd

import (
	"fmt"

	"github.com/provenimpact/wt/internal/shell"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <shell>",
	Short: "Output shell integration function",
	Long:  "Output a shell function that wraps the wt binary to enable directory changing.\n\nSupported shells: bash, zsh, fish\n\nAdd to your shell config:\n  eval \"$(wt init bash)\"   # for .bashrc\n  eval \"$(wt init zsh)\"    # for .zshrc\n  wt init fish | source    # for config.fish",
	Args:  cobra.ExactArgs(1),
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	shellName := args[0]

	code, err := shell.Generate(shellName)
	if err != nil {
		return err
	}

	// Shell function code goes to stdout so it can be eval'd
	fmt.Print(code)
	return nil
}
