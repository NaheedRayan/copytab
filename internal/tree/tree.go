package tree

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxDepth = 5

// BuildTree returns an ASCII tree representation of the directory at root,
// respecting .gitignore rules.
func BuildTree(root string) string {
	ignores := parseGitignore(root)
	var sb strings.Builder
	sb.WriteString(filepath.Base(root) + "\n")
	walkDir(root, "", 0, ignores, &sb)
	return sb.String()
}

type gitignore struct {
	patterns []string
	base     string
}

func parseGitignore(root string) []gitignore {
	var ignores []gitignore

	if data, err := os.ReadFile(filepath.Join(root, ".gitignore")); err == nil {
		ignores = append(ignores, parseGitignoreContent(root, string(data)))
	}

	return ignores
}

func parseGitignoreContent(base, content string) gitignore {
	var patterns []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return gitignore{patterns: patterns, base: base}
}

var skipEntries = map[string]bool{
	".git":         true,
	"node_modules": true,
	".DS_Store":    true,
	"__pycache__":  true,
	".idea":        true,
	".vscode":      true,
}

func isIgnoredForWalk(walkRoot, name string, ignores []gitignore) bool {
	if skipEntries[name] {
		return true
	}

	if strings.HasPrefix(name, ".") && name != ".gitignore" {
		return true
	}

	for _, gi := range ignores {
		rel, err := filepath.Rel(gi.base, filepath.Join(walkRoot, name))
		if err != nil {
			continue
		}
		for _, pattern := range gi.patterns {
			if strings.HasPrefix(pattern, "!") {
				continue
			}
			if matchGitignorePattern(rel, pattern) {
				return true
			}
		}
	}

	return false
}

func matchGitignorePattern(relPath, pattern string) bool {
	pattern = strings.TrimSuffix(pattern, "/")

	if strings.HasPrefix(pattern, "!") {
		return false
	}

	if strings.HasPrefix(pattern, "/") {
		pattern = pattern[1:]
		return gitmatch(relPath, pattern)
	}

	if strings.Contains(pattern, "/") {
		return gitmatch(relPath, pattern) || gitmatch(relPath, "*/"+pattern)
	}

	name := filepath.Base(relPath)
	if gitmatch(name, pattern) {
		return true
	}

	for part := range strings.SplitSeq(relPath, "/") {
		if gitmatch(part, pattern) {
			return true
		}
	}

	return false
}

func gitmatch(name, pattern string) bool {
	if strings.Contains(pattern, "**") {
		return matchDoubleStar(name, pattern)
	}
	matched, _ := filepath.Match(pattern, name)
	return matched
}

func matchDoubleStar(name, pattern string) bool {
	parts := strings.SplitN(pattern, "**", 2)
	prefix := strings.TrimSuffix(parts[0], "/")
	suffix := strings.TrimPrefix(parts[1], "/")

	if parts[0] == "" && parts[1] == "" {
		return true
	}

	if parts[0] == "" {
		return strings.HasSuffix(name, suffix)
	}

	if parts[1] == "" {
		return strings.HasPrefix(name, prefix)
	}

	return strings.HasPrefix(name, prefix) && strings.Contains(name[len(prefix):], "/"+suffix)
}

func walkDir(root, prefix string, depth int, ignores []gitignore, sb *strings.Builder) {
	if depth >= maxDepth {
		return
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}

	var dirs []os.DirEntry
	var files []os.DirEntry
	for _, e := range entries {
		if isIgnoredForWalk(root, e.Name(), ignores) {
			continue
		}
		if e.IsDir() {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

	all := append(dirs, files...)
	for i, entry := range all {
		isLast := i == len(all)-1
		// connector := "├── "
		// if isLast {
		// 	connector = "└── "
		// }
		// childPrefix := "│   "
		// if isLast {
		// 	childPrefix = "    "
		// }

		// With this ASCII-safe version:
		connector := "|-- "
		if isLast {
			connector = "`-- " // or "\\-- "
		}
		childPrefix := "|   "
		if isLast {
			childPrefix = "    "
		}

		sb.WriteString(prefix + connector + entry.Name() + "\n")

		if entry.IsDir() {
			walkDir(filepath.Join(root, entry.Name()), prefix+childPrefix, depth+1, ignores, sb)
		}
	}
}
