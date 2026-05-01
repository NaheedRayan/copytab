package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/shikho/copytab/internal/clipboard"
	"github.com/shikho/copytab/internal/detect"
	"github.com/shikho/copytab/internal/jetbrains"
	"github.com/shikho/copytab/internal/vscode"
)

var supportedIDEs = []string{"vscode", "cursor", "goland", "intellij", "pycharm", "webstorm", "datagrip"}

func main() {
	ideFlag := flag.String("ide", "detect", "IDE to extract tabs from: detect, all, "+strings.Join(supportedIDEs, ", "))
	printFlag := flag.Bool("print", false, "Print tabs to stdout instead of copying to clipboard")
	contentFlag := flag.Bool("content", false, "Copy file contents instead of file paths")
	flag.Parse()

	ides, err := resolveIDEs(*ideFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var allPaths []string
	seen := make(map[string]bool)

	for _, ide := range ides {
		paths, err := getTabs(ide)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to get tabs from %s: %v\n", ide, err)
			continue
		}
		for _, p := range paths {
			if !seen[p] {
				seen[p] = true
				allPaths = append(allPaths, p)
			}
		}
	}

	var output string
	if *contentFlag {
		output = buildContentOutput(allPaths)
	} else {
		output = strings.Join(allPaths, "\n")
	}

	if *printFlag {
		if output != "" {
			fmt.Println(output)
		}
	} else {
		if err := clipboard.Write(output); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if len(allPaths) == 0 {
		fmt.Println("No open tabs found.")
	} else {
		fmt.Printf("Copied %d tabs to clipboard:\n", len(allPaths))
			for _, p := range allPaths {
				fmt.Printf("  - %s\n", p)
			}
	}
}

// buildContentOutput reads each file and formats it with a path header.
func buildContentOutput(paths []string) string {
	var sb strings.Builder
	for i, p := range paths {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString("=== ")
		sb.WriteString(p)
		sb.WriteString(" ===\n")

		data, err := os.ReadFile(p)
		if err != nil {
			sb.WriteString(fmt.Sprintf("(error reading file: %v)\n", err))
			continue
		}
		sb.Write(data)
	}
	return sb.String()
}

func resolveIDEs(ideFlag string) ([]string, error) {
	switch {
	case ideFlag == "detect":
		app, err := detect.FrontmostApp()
		if err != nil {
			return nil, fmt.Errorf("could not detect frontmost IDE: %w", err)
		}
		ide, ok := detect.IDEFromApp(app)
		if !ok {
			return nil, fmt.Errorf("frontmost app '%s' is not a supported IDE\nSupported: %s", app, strings.Join(supportedIDEs, ", "))
		}
		return []string{ide}, nil

	case ideFlag == "all":
		// Collect VS Code-based IDEs
		var ides []string
		for _, ide := range supportedIDEs {
			if _, ok := vscode.AppNames[ide]; ok {
				ides = append(ides, ide)
			}
		}
		// For JetBrains, we let GetAllTabs handle discovery
		return ides, nil // handled specially in getTabs

	default:
		valid := false
		for _, s := range supportedIDEs {
			if ideFlag == s {
				valid = true
				break
			}
		}
		if !valid {
			return nil, fmt.Errorf("unknown IDE '%s'\nSupported: detect, all, %s", ideFlag, strings.Join(supportedIDEs, ", "))
		}
		return []string{ideFlag}, nil
	}
}

func getTabs(ide string) ([]string, error) {
	if ide == "all" {
		// Collect from all JetBrains IDEs
		jbPaths, err := jetbrains.GetAllTabs()
		if err != nil {
			return nil, err
		}
		// Also collect from VS Code-based IDEs
		for _, vscIDE := range []string{"vscode", "cursor"} {
			paths, err := vscode.GetTabs(vscIDE)
			if err != nil {
				continue
			}
			jbPaths = append(jbPaths, paths...)
		}
		return jbPaths, nil
	}

	// Check if it's a VS Code-based IDE
	if _, ok := vscode.AppNames[ide]; ok {
		return vscode.GetTabs(ide)
	}

	// Otherwise treat as JetBrains
	return jetbrains.GetTabs(ide)
}
