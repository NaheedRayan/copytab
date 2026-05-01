package jetbrains

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// IDEPrefixes maps CLI IDE identifiers to directory name prefixes.
var IDEPrefixes = map[string]string{
	"goland":   "GoLand",
	"intellij": "IntelliJ",
	"pycharm":  "PyCharm",
	"webstorm": "WebStorm",
	"datagrip": "DataGrip",
}

// processNames maps CLI IDE identifiers to macOS process names used by AppleScript.
var processNames = map[string]string{
	"goland":   "GoLand",
	"intellij": "IntelliJ IDEA",
	"pycharm":  "PyCharm",
	"webstorm": "WebStorm",
	"datagrip": "DataGrip",
}

// projectInfo holds a project's path and its workspace ID.
type projectInfo struct {
	path        string
	workspaceID string
	timestamp   int64
	opened      bool
}

// GetTabs extracts open file paths from the specified JetBrains IDE.
func GetTabs(ide string) ([]string, error) {
	prefix, ok := IDEPrefixes[ide]
	if !ok {
		return nil, fmt.Errorf("unsupported JetBrains IDE: %s", ide)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	jetbrainsBase := filepath.Join(home, "Library", "Application Support", "JetBrains")
	entries, err := os.ReadDir(jetbrainsBase)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read JetBrains directory: %w", err)
	}

	// Find the most recent version directory matching the prefix
	var versionDirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), prefix) {
			versionDirs = append(versionDirs, e.Name())
		}
	}
	if len(versionDirs) == 0 {
		return nil, nil
	}
	sort.Sort(sort.Reverse(sort.StringSlice(versionDirs)))

	ideDir := filepath.Join(jetbrainsBase, versionDirs[0])
	projects, err := loadRecentProjects(ideDir, home)
	if err != nil {
		return nil, fmt.Errorf("failed to load recent projects: %w", err)
	}

	// If a project is currently opened, only use that project.
	// Otherwise fall back to the most recently activated project.
	var activeProject *projectInfo
	for i := range projects {
		if projects[i].opened {
			activeProject = &projects[i]
			break
		}
	}
	if activeProject == nil {
		sort.Slice(projects, func(i, j int) bool {
			return projects[i].timestamp > projects[j].timestamp
		})
		if len(projects) > 0 {
			activeProject = &projects[0]
		}
	}

	if activeProject == nil {
		return nil, nil
	}

	// Try to trigger a workspace save via AppleScript so the file on disk
	// reflects the current open tabs. Silently ignores if permissions are
	// not granted.
	triggerSave(ide)

	workspacePath := filepath.Join(ideDir, "workspace", activeProject.workspaceID+".xml")
	return extractTabsFromWorkspace(workspacePath, activeProject.path)
}

// triggerSave uses AppleScript to activate the IDE and click "Save All" in the
// File menu. This flushes the workspace XML to disk so we get live tab data.
// Requires Terminal (or the running shell) to have Accessibility permissions.
func triggerSave(ide string) {
	procName, ok := processNames[ide]
	if !ok {
		return
	}

	script := fmt.Sprintf(
		`tell application "%s" to activate
delay 0.3
tell application "System Events" to tell process "%s" to click menu item "Save All" of menu "File" of menu bar 1`,
		procName, procName,
	)

	cmd := exec.Command("osascript", "-e", script)
	cmd.Run()

	// Give the IDE a moment to write the workspace file
	time.Sleep(500 * time.Millisecond)
}

// GetAllTabs extracts tabs from all installed JetBrains IDEs.
func GetAllTabs() ([]string, error) {
	var allPaths []string
	seen := make(map[string]bool)

	for ide := range IDEPrefixes {
		paths, err := GetTabs(ide)
		if err != nil {
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

func loadRecentProjects(ideDir, home string) ([]projectInfo, error) {
	path := filepath.Join(ideDir, "options", "recentProjects.xml")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	return parseRecentProjects(f, home)
}

func parseRecentProjects(r io.Reader, home string) ([]projectInfo, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	decoder.Strict = false

	var app struct {
		Components []struct {
			Name string `xml:"name,attr"`
			Raw  string `xml:",innerxml"`
		} `xml:"component"`
	}
	if err := decoder.Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to parse recentProjects.xml: %w", err)
	}

	var projects []projectInfo
	for _, comp := range app.Components {
		if comp.Name != "RecentProjectsManager" {
			continue
		}

		decoder := xml.NewDecoder(strings.NewReader(comp.Raw))
		decoder.Strict = false

		type mapEntry struct {
			Key   string `xml:"key,attr"`
			Value struct {
				Inner string `xml:",innerxml"`
			} `xml:"value"`
		}
		type mapType struct {
			Entries []mapEntry `xml:"entry"`
		}

		for {
			tok, err := decoder.Token()
			if err == io.EOF || tok == nil {
				break
			}
			if err != nil {
				break
			}

			se, ok := tok.(xml.StartElement)
			if !ok || se.Name.Local != "map" {
				continue
			}

			var m mapType
			if err := decoder.DecodeElement(&m, &se); err != nil {
				continue
			}

			for _, entry := range m.Entries {
				projPath := strings.ReplaceAll(entry.Key, "$USER_HOME$", home)

				wsID, ts, opn := parseMetaInfo(entry.Value.Inner)

				projects = append(projects, projectInfo{
					path:        projPath,
					workspaceID: wsID,
					timestamp:   ts,
					opened:      opn,
				})
			}
		}
		break
	}

	return projects, nil
}

// parseMetaInfo extracts projectWorkspaceId, activationTimestamp, and opened
// from the inner XML of a RecentProjectMetaInfo element.
func parseMetaInfo(innerXML string) (workspaceID string, timestamp int64, opened bool) {
	decoder := xml.NewDecoder(strings.NewReader(innerXML))
	decoder.Strict = false

	for {
		tok, err := decoder.Token()
		if err == io.EOF || tok == nil {
			break
		}
		if err != nil {
			break
		}

		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		switch se.Name.Local {
		case "RecentProjectMetaInfo":
			for _, attr := range se.Attr {
				if attr.Name.Local == "projectWorkspaceId" {
					workspaceID = attr.Value
				}
				if attr.Name.Local == "opened" && attr.Value == "true" {
					opened = true
				}
			}
		case "option":
			for _, attr := range se.Attr {
				if attr.Name.Local == "name" && attr.Value == "activationTimestamp" {
					var opt struct {
						Value string `xml:"value,attr"`
					}
					if decoder.DecodeElement(&opt, &se) == nil {
						timestamp, _ = strconv.ParseInt(opt.Value, 10, 64)
					}
				}
			}
		}
	}

	return workspaceID, timestamp, opened
}

func extractTabsFromWorkspace(workspacePath, projectPath string) ([]string, error) {
	f, err := os.Open(workspacePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	content := string(data)

	femIdx := strings.Index(content, `name="FileEditorManager"`)
	if femIdx == -1 {
		return nil, nil
	}

	section := content[femIdx:]
	nextComp := strings.Index(section[1:], `<component name="`)
	if nextComp != -1 {
		section = section[:nextComp+1]
	}

	var paths []string
	seen := make(map[string]bool)
	extractEntryPaths(section, projectPath, seen, &paths)

	return paths, nil
}

func extractEntryPaths(fragment, projectPath string, seen map[string]bool, paths *[]string) {
	for {
		idx := strings.Index(fragment, `entry file="`)
		if idx == -1 {
			break
		}
		fragment = fragment[idx+len(`entry file="`):]
		endQuote := strings.Index(fragment, `"`)
		if endQuote == -1 {
			break
		}
		rawPath := fragment[:endQuote]

		resolved := resolveFilePath(rawPath, projectPath)
		if resolved != "" && !seen[resolved] {
			seen[resolved] = true
			*paths = append(*paths, resolved)
		}
	}
}

func resolveFilePath(raw, projectPath string) string {
	if !strings.HasPrefix(raw, "file://") {
		return ""
	}
	trimmed := strings.TrimPrefix(raw, "file://")
	resolved := strings.ReplaceAll(trimmed, "$PROJECT_DIR$", projectPath)
	return resolved
}
