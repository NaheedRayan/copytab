package content

import (
	"fmt"
	"os"
	"strings"
)

// BuildOutput reads each file and formats it with a path header.
func BuildOutput(paths []string) string {
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
