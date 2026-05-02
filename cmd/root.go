package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/NaheedRayan/copytab/internal/clipboard"
	"github.com/NaheedRayan/copytab/internal/content"
	"github.com/NaheedRayan/copytab/internal/tree"
	"github.com/spf13/cobra"
)

var (
	ideFlag   string
	printFlag bool
	treeFlag  bool
)

var rootCmd = &cobra.Command{
	Use:   "copytab",
	Short: "Copy open IDE tab contents to your clipboard",
	Long:  "One-shot your LLM with only the content that matters.\nDetects your frontmost IDE and copies all open file contents to your clipboard.",
	RunE:  runCopyContents,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&ideFlag, "ide", "detect",
		"IDE to extract tabs from: detect, all, "+strings.Join(supportedIDEs, ", "))
	rootCmd.PersistentFlags().BoolVar(&printFlag, "print", false,
		"Print to stdout instead of copying to clipboard")
	rootCmd.PersistentFlags().BoolVar(&treeFlag, "tree", false,
		"Include folder structure tree in the output")
}

func runCopyContents(cmd *cobra.Command, args []string) error {
	allPaths, err := collectPaths()
	if err != nil {
		return err
	}

	output := content.BuildOutput(allPaths)

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

	printSummary(allPaths, "contents")
	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
