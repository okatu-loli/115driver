# 115driver CLI Design Spec

## Overview

Add a CLI tool to 115driver for operating 115 cloud storage from the terminal. Designed for both human users (table/colored output) and AI agents (JSON output via `--json` flag). The CLI runs alongside the existing MCP server, sharing the core `pkg/driver/` library.

## Architecture

### Approach: Independent CLI entry + shared driver library

```
115driver/
├── pkg/driver/     # Core library (unchanged)
├── pkg/crypto/     # Crypto utilities (unchanged)
├── mcp/            # MCP server (unchanged)
└── cli/            # NEW: CLI tool
    ├── main.go
    ├── cmd/        # Cobra subcommands
    └── internal/   # Internal packages
        ├── auth/   # Authentication management
        └── output/ # Output formatting
```

CLI and MCP are parallel consumers of `pkg/driver/`. They do not depend on each other.

### Dependencies

- **cobra** — CLI framework (subcommands, flags, help generation, shell completion)
- **viper** — Configuration management (TOML config, env vars, flag binding)
- **tablewriter** — Table rendering for human-readable output
- **cheggaaa/pb/v3** — Progress bar for upload/download
- **pkg/driver/** — All 115 cloud operations (file, dir, search, offline, upload, download)

## Path Resolution

### Problem

All `pkg/driver/` methods operate on **file/directory IDs** (numeric strings), not human-readable paths. CLI users expect to type paths like `/documents/report.pdf`.

### Solution: Use existing `DirName2CID` API

Driver already provides `DirName2CID(dir string)` which converts a path string to directory ID. CLI adds a `resolvePath` helper in `cli/internal/resolver/`:

```go
// resolvePath converts a remote path to a file/directory ID.
// For directories: uses driver.DirName2CID()
// For files: lists parent directory via driver.List(), matches by name
func resolvePath(client *Pan115Client, remotePath string) (fileID string, isDir bool, error)
```

### Resolution strategy

1. **Directory paths** (`/documents/projects/`): call `DirName2CID("documents/projects")` directly — one API call.
2. **File paths** (`/documents/report.pdf`): call `DirName2CID("documents")` to get parent ID, then `List(parentID)` and match by filename.
3. **Root path** (`/`): ID is `"0"`.
4. **No caching in v1**: each command resolves paths independently. If performance becomes an issue, add an LRU cache later.

### Path conventions

- Paths starting with `/` are absolute (from root).
- Paths without `/` are relative to root (equivalent to `/<path>`).
- Driver's `DirName2CID` trims leading `/` internally.

## Command Design

### Global Flags

| Flag | Env Variable | Description |
|------|-------------|-------------|
| `--cookie` | `115DRIVER_COOKIE` | Cookie string |
| `--config` | `115DRIVER_CONFIG` | Config file path (default `~/.115driver/config.toml`) |
| `--profile` | `115DRIVER_PROFILE` | Config profile name (default `main`) |
| `--json` | — | JSON output for AI agents |
| `--debug` | `115DRIVER_DEBUG` | Debug mode with verbose logging |

### Authentication Commands

```
115driver login                          # Interactive QR code login
115driver login --cookie="UID=xxx;..."   # Cookie-based login
115driver whoami                         # Show current auth user info
```

**JSON mode behavior for auth commands:**
- `login --json`: outputs QR code session URL (text) instead of terminal QR image. Polls and returns `{"success": true, "data": {"cookie_set": true}}` on success.
- `whoami --json`: `{"success": true, "data": {"user_id": 123, "username": "..."}}`

### File & Directory Commands

```
115driver ls [remote_path]               # List directory (default /)
115driver ls /path -l                    # Detailed listing with sizes/dates
115driver stat <remote_path>             # File/directory details
115driver mkdir [-p] <remote_path>       # Create directory (-p: create parents)
115driver rename <remote_path> <name>    # Rename file or directory
115driver mv <src_path> <dst_dir_path>   # Move file(s) into target directory
115driver cp <src_path> <dst_dir_path>   # Copy file(s) into target directory
115driver rm <remote_path>               # Move to recycle bin (default)
115driver rm -f <remote_path>            # Permanently delete (skip recycle bin)
```

**`mv` and `cp` semantics** (aligned with driver API):

Both `Move(dirID, fileIDs...)` and `Copy(dirID, fileIDs...)` accept a **target directory ID** and one or more **source file IDs**. They move/copy files *into* the target directory, preserving original names. The driver API does not support rename-during-move.

- `mv /docs/a.pdf /backup/` → resolve `/docs/a.pdf` to fileID, resolve `/backup/` to dirID, call `Move(dirID, fileID)`
- `cp /docs/a.pdf /backup/` → same pattern with `Copy`
- If dst is a file that already exists: **error** (driver API does not support overwrite)
- If dst is a directory: file is moved/copied into it
- Trailing `/` on dst is optional but recommended for clarity

**`rm` semantics:**

- Default (`rm <path>`): calls `driver.Delete(fileIDs...)` which moves files to the **recycle bin** (115's server-side behavior).
- `-f` flag: **not yet supported** in v1 — permanent deletion requires a separate clean-recycle-bin API with password. Flag reserved for future use.
- `-r` flag: not needed — `Delete` works on both files and directories.
- No confirmation prompt in JSON mode. Human mode prompts when deleting directories.

### Upload & Download Commands

```
115driver upload <local_path> <remote_dir>    # Upload file to remote directory
115driver download <remote_path> <local_dir>  # Download file to local directory
```

- Upload uses `UploadFastOrByOSS` / `RapidUploadOrByOSS` (SHA1 dedup + OSS fallback).
- Download resolves remote path to pickCode, then calls `driver.Download(pickCode)`.
- Both show progress bar in human mode.

### Search Command

```
115driver search <keyword>                    # Search files
115driver search <keyword> -t video           # Filter by type
115driver search <keyword> --sort name        # Sort results
```

Type filters: `folder`, `document`, `image`, `video`, `audio`, `archive`

Uses `driver.Search(opts)` directly — no path resolution needed.

### Offline Download Commands

```
115driver offline add <url> [-d <remote_dir>] # Add offline task (http/ed2k/magnet)
115driver offline list                        # List offline tasks
115driver offline rm <hash>                   # Remove offline task by info hash
```

- `offline add` defaults to root directory (`"0"`). Use `-d` to specify save directory.
- Uses `driver.AddOfflineTaskURIs(uris, saveDirID)`.
- `offline rm` uses `driver.DeleteOfflineTasks(hashes, deleteFiles=false)`.

### Share Commands

**Removed from v1.** Driver only has `GetShareSnap` (read share info), no create-share API. Share commands deferred until driver support is added.

## Authentication System

### Credential Priority (highest to lowest)

This priority chain applies to **non-login commands** only:

1. `--cookie` flag
2. `115DRIVER_COOKIE` environment variable
3. Config file profile (`~/.115driver/config.toml`)

If none of the above provide a valid credential, the command **exits with code 2** (auth error) and suggests running `115driver login`. Commands never auto-trigger interactive login.

### Login command (separate flow)

`115driver login` is the only command that performs interactive authentication:

- **QR code flow**: calls `driver.QRCodeStart()` → displays QR → polls `driver.QRCodeStatus()` → calls `driver.QRCodeLogin()` → saves cookie to config file
- **Cookie flow**: `login --cookie="..."` → calls `driver.LoginCheck()` to validate → saves to config file

### Config File Format (TOML)

```toml
# ~/.115driver/config.toml

default_profile = "main"

[profiles.main]
cookie = "UID=xxx;CID=xxx;SEID=xxx;KID=xxx"

[profiles.work]
cookie = "UID=yyy;CID=yyy;SEID=yyy;KID=yyy"
```

Multi-profile support allows account switching via `--profile work`.

## Output System

### Dual-mode output

**Human mode** (default):
- `ls`: table with columns (name, size, date, type)
- `stat`: labeled key-value pairs
- Operation results: concise success/failure messages
- Upload/download: progress bar (cheggaaa/pb/v3)
- Errors: colored messages with remediation hints

**JSON mode** (`--json`):
- Structured JSON for AI agent consumption
- All output goes to stdout as valid JSON
- Errors also JSON-formatted to stdout (no stderr noise)

### JSON Output Schema

Every command returns a consistent envelope:

```json
{
  "success": true|false,
  "data": { ... },    // present on success
  "error": "...",     // present on failure
  "code": 0           // exit code equivalent
}
```

**Per-command data schemas:**

`ls`:
```json
{"success": true, "data": {"path": "/docs", "files": [
  {"name": "report.pdf", "size": 1024, "is_dir": false, "update_time": "2026-04-16T10:00:00Z", "file_id": "12345", "pick_code": "abc"},
  {"name": "projects", "size": 0, "is_dir": true, "update_time": "2026-04-15T08:00:00Z", "file_id": "67890"}
]}}
```

`stat`:
```json
{"success": true, "data": {"name": "report.pdf", "size": 1024, "is_dir": false, "sha1": "abc123...", "pick_code": "xyz", "create_time": "...", "update_time": "...", "parents": [{"id": "100", "name": "docs"}]}}
```

`mkdir`:
```json
{"success": true, "data": {"name": "newdir", "dir_id": "55555", "path": "/docs/newdir"}}
```

`rename`:
```json
{"success": true, "data": {"old_name": "a.pdf", "new_name": "b.pdf", "file_id": "12345"}}
```

`mv` / `cp`:
```json
{"success": true, "data": {"source": "/docs/a.pdf", "destination_dir": "/backup/", "file_ids": ["12345"]}}
```

`rm`:
```json
{"success": true, "data": {"deleted": ["/docs/a.pdf"], "file_ids": ["12345"]}}
```

`upload`:
```json
{"success": true, "data": {"local_path": "./report.pdf", "remote_dir": "/docs/", "method": "rapid", "size": 1024}}
```

`download`:
```json
{"success": true, "data": {"remote_path": "/docs/report.pdf", "local_path": "./report.pdf", "size": 1024}}
```

`search`:
```json
{"success": true, "data": {"keyword": "report", "count": 2, "files": [
  {"name": "report.pdf", "size": 1024, "is_dir": false, "file_id": "12345"}
]}}
```

`offline add`:
```json
{"success": true, "data": {"url": "magnet:?xt=...", "hashes": ["abc123"], "save_dir": "/"}}
```

`offline list`:
```json
{"success": true, "data": {"total": 5, "tasks": [
  {"name": "file.mkv", "hash": "abc123", "status": "running", "percent": 45.2, "size": 1073741824}
]}}
```

`offline rm`:
```json
{"success": true, "data": {"deleted_hashes": ["abc123"]}}
```

`whoami`:
```json
{"success": true, "data": {"user_id": 12345, "username": "user@example"}}
```

`login` (success):
```json
{"success": true, "data": {"profile": "main", "cookie_saved": true}}
```

### Output Module Structure

```
cli/internal/output/
├── printer.go     # Printer interface + factory (NewPrinter(json bool))
├── table.go       # Table output for human mode
├── json.go        # JSON output for AI mode
└── progress.go    # Progress bar for uploads/downloads
```

## Error Handling

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Authentication failure (expired/invalid cookie) |
| 3 | File/directory not found |
| 4 | Network error |
| 5 | Invalid arguments |

**Interaction with Cobra**: Cobra handles flag parsing errors and unknown command errors. Root command's `SilenceErrors = true` and `SilenceUsage = true` suppress Cobra's default error output. All errors are routed through CLI's own error handler which sets the correct exit code via `os.Exit()` in `main()`. Cobra's `RunE` returns CLI-defined error types that carry the exit code.

### Error Behavior

- Cookie expired: error message suggests `115driver login`
- Network timeout: automatic single retry
- Path not found: exit code 3, show closest existing parent path
- JSON mode: all errors formatted as `{"success": false, "error": "...", "code": N}` to stdout

## File Structure

```
cli/
├── main.go                    # Entry point: create root command, Execute(), handle exit codes
├── cmd/
│   ├── root.go                # Root command, global flags, PersistentPreRunE (auth init)
│   ├── login.go               # Login command (QR code + cookie)
│   ├── whoami.go              # Current user info
│   ├── ls.go                  # List directory
│   ├── stat.go                # File/directory details
│   ├── mkdir.go               # Create directory
│   ├── rename.go              # Rename
│   ├── mv.go                  # Move into directory
│   ├── cp.go                  # Copy into directory
│   ├── rm.go                  # Delete (to recycle bin)
│   ├── upload.go              # Upload file
│   ├── download.go            # Download file
│   ├── search.go              # Search files
│   └── offline.go             # Offline download management (add/list/rm subcommands)
├── internal/
│   ├── auth/
│   │   └── auth.go            # Credential resolution (flag > env > config) + config file I/O
│   ├── output/
│   │   ├── printer.go         # Printer interface + envelope struct
│   │   ├── table.go           # Human-readable table output
│   │   ├── json.go            # JSON output
│   │   └── progress.go        # Progress bar wrapper
│   └── resolver/
│       └── resolver.go        # Path→ID resolution using DirName2CID + List
└── (part of main go module)
```

## Key Design Decisions

1. **Same Go module**: CLI lives under `cli/` within the main module (`go build -o 115driver ./cli/`). No separate `go.mod`, no `go.work`. MCP server builds separately: `go build -o 115driver-mcp-server ./mcp/`.

2. **Build target**: `go build -o 115driver ./cli/` produces the CLI binary.

3. **No shared code between CLI and MCP**: They both use `pkg/driver/` directly. If shared auth logic is needed later, it moves to `pkg/driver/` or a new `pkg/auth/`.

4. **Path resolution via DirName2CID**: Driver already provides this API. CLI wraps it with file-name matching for non-directory paths.

5. **Operation alignment with driver**: CLI commands map 1:1 to `pkg/driver/` methods. `mv` and `cp` accept target *directory* (not rename target) to match driver API. No CLI-only business logic.

6. **Share commands deferred**: Driver lacks create-share API. Removed from v1 scope.

7. **rm defaults to recycle bin**: 115's `Delete` API moves to recycle bin by default. `-f` flag reserved for future permanent-delete support.
