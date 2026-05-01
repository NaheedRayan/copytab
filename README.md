# copytab

CLI tool that copies all open IDE tabs to your clipboard on macOS.

## Supported IDEs

- VS Code / Cursor
- GoLand / IntelliJ IDEA / PyCharm / WebStorm / DataGrip

## Install

```bash
go install github.com/NaheedRayan/copytab@latest
```



Make sure `~/go/bin` is in your `PATH`:

```bash
export PATH="$HOME/go/bin:$PATH"
```

## Run from this directory

```bash
go run .
```

## Setup

### Accessibility permission (required for JetBrains IDEs)

JetBrains IDEs cache workspace state to disk and don't flush it on every tab change. To get **live** tab data, `copytab` triggers a "Save All" via AppleScript before reading the workspace file.

This requires granting **Accessibility** permission to your terminal app:

1. Open **System Settings > Privacy & Security > Accessibility**
2. Click the **+** button
3. Add your terminal app (e.g. **Terminal**, **iTerm2**, **Warp**, **VS Code**)

Without this permission, JetBrains tabs will still work but may reflect a slightly stale state.

## Usage

```bash
copytab                        # Auto-detect frontmost IDE, copy tab paths to clipboard
copytab --ide=vscode           # Copy VS Code tab paths
copytab --ide=goland           # Copy GoLand tab paths
copytab --ide=all              # Collect from all IDEs, deduplicate, copy to clipboard
copytab --content              # Copy file contents instead of file paths
copytab --print                # Print to stdout instead of clipboard
copytab --ide=goland --content --print  # Print GoLand file contents to stdout
```

### Flags

| Flag | Description |
|------|-------------|
| `--ide` | IDE to extract tabs from: `detect`, `all`, or a specific IDE name |
| `--content` | Copy file contents instead of file paths |
| `--print` | Print to stdout instead of copying to clipboard |

### IDE names

`vscode`, `cursor`, `goland`, `intellij`, `pycharm`, `webstorm`, `datagrip`

## How it works

**VS Code / Cursor** — Reads the SQLite database (`state.vscdb`) from the most recently used workspace under `~/Library/Application Support/{Code,Cursor}/User/workspaceStorage/<hash>/`. The open tabs are stored in a `memento/workbench.parts.editor` key as JSON. VS Code updates this database in real-time, so tab data is always current.

**JetBrains IDEs** — Triggers a "Save All" via AppleScript to flush workspace state to disk (requires Accessibility permission). Then parses `recentProjects.xml` to find the currently open project (`opened="true"`), and reads its workspace XML file to extract open file entries from the `FileEditorManager` component.

**Clipboard** — Uses macOS `pbcopy`.

## Debugging

Print tabs to stdout instead of copying to clipboard:

```bash
go run . --print
go run . --ide=goland --print
go run . --ide=goland --content --print
go run . --ide=vscode --print
```

Check what the tool sees for a specific IDE:

```bash
# VS Code workspace databases
ls -lt ~/Library/Application\ Support/Code/User/workspaceStorage/

# JetBrains workspace files
ls -lt ~/Library/Application\ Support/JetBrains/GoLand*/workspace/

# JetBrains recent projects mapping
cat ~/Library/Application\ Support/JetBrains/GoLand*/options/recentProjects.xml
```

## Build

```bash
go build -o copytab .
./copytab --ide=vscode --print
```
