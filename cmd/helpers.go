package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/NaheedRayan/copytab/internal/detect"
	"github.com/NaheedRayan/copytab/internal/jetbrains"
	"github.com/NaheedRayan/copytab/internal/vscode"
)

var supportedIDEs = []string{"vscode", "cursor", "goland", "intellij", "pycharm", "webstorm", "datagrip"}

func collectPaths() ([]string, error) {
	ides, err := resolveIDEs(ideFlag)
	if err != nil {
		return nil, err
	}

	var allPaths []string
	seen := make(map[string]bool)

	for _, ide := range ides {
		var paths []string

		switch {
		case isVSCodeIDE(ide):
			paths, err = vscode.GetTabs(ide)
		case isJetBrainsIDE(ide):
			paths, err = jetbrains.GetTabs(ide)
		default:
			err = fmt.Errorf("unknown IDE: %s", ide)
		}

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

	return allPaths, nil
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
			return nil, fmt.Errorf("frontmost app '%s' is not a supported IDE\nSupported: %s",
				app, strings.Join(supportedIDEs, ", "))
		}
		return []string{ide}, nil

	case ideFlag == "all":
		return supportedIDEs, nil

	default:
		for _, s := range supportedIDEs {
			if ideFlag == s {
				return []string{ideFlag}, nil
			}
		}
		return nil, fmt.Errorf("unknown IDE '%s'\nSupported: detect, all, %s",
			ideFlag, strings.Join(supportedIDEs, ", "))
	}
}

func isVSCodeIDE(ide string) bool {
	_, ok := vscode.AppNames[ide]
	return ok
}

func isJetBrainsIDE(ide string) bool {
	_, ok := jetbrains.IDEPrefixes[ide]
	return ok
}

func printSummary(paths []string, mode string) {
	if len(paths) == 0 {
		fmt.Println("No open tabs found.")
		return
	}
	if !printFlag {
		fmt.Printf("Copied %d tab %s to clipboard:\n", len(paths), mode)
	} else {
		fmt.Printf("Collected %d tab %s:\n", len(paths), mode)
	}
	for _, p := range paths {
		fmt.Printf("  - %s\n", p)
	}
}
