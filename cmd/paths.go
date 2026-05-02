package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/NaheedRayan/copytab/internal/clipboard"
	"github.com/NaheedRayan/copytab/internal/tree"
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

	if treeFlag {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not determine working directory: %w", err)
		}
		treeOutput := tree.BuildTree(wd)
		output = "=== Folder Structure ===\n" + treeOutput + "\n" + output
	}

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
