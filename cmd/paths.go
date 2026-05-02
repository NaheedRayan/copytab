package cmd

import (
	"fmt"
	"strings"

	"github.com/NaheedRayan/copytab/internal/clipboard"
	"github.com/spf13/cobra"
)

var pathsCmd = &cobra.Command{
	Use:   "paths",
	Short: "Copy open IDE tab file paths to your clipboard",
	RunE:  runCopyPaths,
}

func init() {
	rootCmd.AddCommand(pathsCmd)
}

func runCopyPaths(cmd *cobra.Command, args []string) error {
	allPaths, err := collectPaths()
	if err != nil {
		return err
	}

	output := strings.Join(allPaths, "\n")

	if printFlag {
		if output != "" {
			fmt.Println(output)
		}
	} else {
		if err := clipboard.Write(output); err != nil {
			return err
		}
	}

	printSummary(allPaths, "paths")
	return nil
}
