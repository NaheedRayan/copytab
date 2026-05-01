package detect

import (
	"fmt"
	"os/exec"
	"strings"
)

// AppToIDE maps macOS application names (as returned by osascript) to IDE identifiers.
var AppToIDE = map[string]string{
	"Code":               "vscode",
	"Visual Studio Code": "vscode",
	"Cursor":             "cursor",
	"GoLand":             "goland",
	"IntelliJ IDEA":      "intellij",
	"PyCharm":            "pycharm",
	"WebStorm":           "webstorm",
	"DataGrip":           "datagrip",
}

// FrontmostApp returns the name of the currently focused application.
func FrontmostApp() (string, error) {
	cmd := exec.Command("osascript", "-e", `tell application "System Events" to set frontApp to name of first application process whose frontmost is true`)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to detect frontmost app: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// IDEFromApp maps a macOS application name to a standardized IDE identifier.
// Returns the IDE name and true if recognized, empty string and false otherwise.
func IDEFromApp(appName string) (string, bool) {
	ide, ok := AppToIDE[appName]
	return ide, ok
}
