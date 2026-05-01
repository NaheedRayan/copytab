package clipboard

import (
	"fmt"
	"os/exec"
	"strings"
)

// Write copies the given text to the macOS clipboard using pbcopy.
func Write(text string) error {
	if text == "" {
		return nil
	}
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}
	return nil
}
