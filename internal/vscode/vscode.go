package vscode

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

// AppNames maps IDE identifiers to their macOS application support directory names.
var AppNames = map[string]string{
	"vscode": "Code",
	"cursor": "Cursor",
}

// GetTabs extracts open file paths from the specified VS Code-based IDE.
// ide must be "vscode" or "cursor".
func GetTabs(ide string) ([]string, error) {
	appName, ok := AppNames[ide]
	if !ok {
		return nil, fmt.Errorf("unsupported IDE: %s", ide)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	basePath := filepath.Join(home, "Library", "Application Support", appName, "User", "workspaceStorage")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read workspace storage: %w", err)
	}

	// Sort by modification time, most recent first.
	// Only use the most recently modified workspace — it's the active one
	// and its state.vscdb is updated in real-time when tabs change.
	type dirInfo struct {
		path string
		mod  int64
	}
	var dirs []dirInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		fullPath := filepath.Join(basePath, e.Name())
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		dirs = append(dirs, dirInfo{path: fullPath, mod: info.ModTime().UnixMicro()})
	}
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].mod > dirs[j].mod
	})

	if len(dirs) == 0 {
		return nil, nil
	}

	// Only read the most recently modified workspace
	dbPath := filepath.Join(dirs[0].path, "state.vscdb")
	paths, err := extractTabsFromDB(dbPath)
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func extractTabsFromDB(dbPath string) ([]string, error) {
	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var rawValue string
	err = db.QueryRow("SELECT value FROM ItemTable WHERE key = 'memento/workbench.parts.editor'").Scan(&rawValue)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return parseEditorMemento(rawValue)
}

func parseEditorMemento(raw string) ([]string, error) {
	var memento map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &memento); err != nil {
		return nil, err
	}

	editorStateRaw, ok := memento["editorpart.state"]
	if !ok {
		return nil, nil
	}

	var editorState struct {
		SerializedGrid struct {
			Root node `json:"root"`
		} `json:"serializedGrid"`
	}
	if err := json.Unmarshal(editorStateRaw, &editorState); err != nil {
		return nil, err
	}

	return collectFilePaths(editorState.SerializedGrid.Root), nil
}

type node struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type leafData struct {
	Editors []editorEntry `json:"editors"`
}

type editorEntry struct {
	ID    string          `json:"id"`
	Value json.RawMessage `json:"value"`
}

func collectFilePaths(n node) []string {
	if n.Type == "leaf" {
		var ld leafData
		if err := json.Unmarshal(n.Data, &ld); err != nil {
			return nil
		}
		var paths []string
		for _, ed := range ld.Editors {
			p := extractFilePath(ed)
			if p != "" {
				paths = append(paths, p)
			}
		}
		return paths
	}

	var children []node
	if err := json.Unmarshal(n.Data, &children); err != nil {
		return nil
	}

	var paths []string
	for _, child := range children {
		paths = append(paths, collectFilePaths(child)...)
	}
	return paths
}

func extractFilePath(ed editorEntry) string {
	if !strings.Contains(ed.ID, "fileEditorInput") {
		return ""
	}

	var rawStr string
	if err := json.Unmarshal(ed.Value, &rawStr); err != nil {
		return ""
	}

	var val struct {
		ResourceJSON struct {
			FsPath string `json:"fsPath"`
		} `json:"resourceJSON"`
	}
	if err := json.Unmarshal([]byte(rawStr), &val); err != nil {
		return ""
	}
	return val.ResourceJSON.FsPath
}
