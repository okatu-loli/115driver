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
- **pkg/driver/** — All 115 cloud operations (file, dir, search, offline, share, upload, download)

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

### File & Directory Commands

```
115driver ls [remote_path]               # List directory (default /)
115driver ls /path -l                    # Detailed listing with sizes/dates
115driver stat <remote_path>             # File/directory details
115driver mkdir [-p] <remote_path>       # Create directory (-p: create parents)
115driver rename <remote_path> <name>    # Rename file or directory
115driver mv <src> <dst>                 # Move file(s)
115driver cp <src> <dst>                 # Copy file(s)
115driver rm [-r] [-f] <remote_path>     # Delete file or directory
```

### Upload & Download Commands

```
115driver upload <local_path> <remote_dir>    # Upload file
115driver download <remote_path> <local_dir>  # Download file
```

### Search Command

```
115driver search <keyword>                    # Search files
115driver search <keyword> -t video           # Filter by type
115driver search <keyword> --sort name        # Sort results
```

Type filters: `folder`, `document`, `image`, `video`, `audio`, `archive`

### Offline Download Commands

```
115driver offline add <url>                   # Add offline task (http/ed2k/magnet)
115driver offline list                        # List offline tasks
115driver offline rm <task_id>                # Remove offline task
```

### Share Commands

```
115driver share create <file_id>              # Create share link
115driver share list                          # List shares
```

## Authentication System

### Credential Priority (highest to lowest)

1. `--cookie` flag
2. `115DRIVER_COOKIE` environment variable
3. Config file profile (`~/.115driver/config.toml`)
4. Interactive QR code login

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

### Login Flow

- `115driver login`: calls `pkg/driver/` QR code login, saves resulting cookie to config file
- `115driver login --cookie="..."`: validates cookie, saves to config file
- All subsequent commands read cookie from config file automatically

## Output System

### Dual-mode output

**Human mode** (default):
- `ls`: table with columns (name, size, date, type)
- `stat`: labeled key-value pairs
- Operation results: concise success/failure messages
- Upload/download: progress bar
- Errors: colored messages with remediation hints

**JSON mode** (`--json`):
- Structured JSON for AI agent consumption
- Success: `{"success": true, "data": {...}}`
- Error: `{"success": false, "error": "message", "code": 2}`
- No stderr noise — all output goes to stdout as valid JSON

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

### Error Behavior

- Cookie expired: prompt user to run `115driver login`
- Network timeout: automatic single retry
- JSON mode: errors formatted as JSON, no stderr noise

## File Structure

```
cli/
├── main.go                    # Entry point
├── cmd/
│   ├── root.go                # Root command, global flags, PersistentPreRun auth
│   ├── login.go               # Login command
│   ├── whoami.go              # Current user info
│   ├── ls.go                  # List directory
│   ├── stat.go                # File/directory details
│   ├── mkdir.go               # Create directory
│   ├── rename.go              # Rename
│   ├── mv.go                  # Move
│   ├── cp.go                  # Copy
│   ├── rm.go                  # Delete
│   ├── upload.go              # Upload file
│   ├── download.go            # Download file
│   ├── search.go              # Search files
│   ├── offline.go             # Offline download management
│   └── share.go               # Share link management
├── internal/
│   ├── auth/
│   │   └── auth.go            # Credential resolution (flag > env > config > interactive)
│   └── output/
│       ├── printer.go         # Printer interface
│       ├── table.go           # Human-readable table output
│       ├── json.go            # JSON output
│       └── progress.go        # Progress bar
└── go.mod                     # Part of main module
```

## Key Design Decisions

1. **Same Go module**: CLI lives under `cli/` within the main module, not a separate Go module. This avoids version drift with `pkg/driver/`.

2. **Build target**: `go build -o 115driver ./cli/` produces the CLI binary. MCP server builds separately: `go build -o 115driver-mcp-server ./mcp/`.

3. **No shared code between CLI and MCP**: They both use `pkg/driver/` directly. If shared auth logic is needed later, it moves to `pkg/driver/` or a new `pkg/auth/`.

4. **Remote path convention**: Remote paths start with `/` (e.g., `/documents/file.pdf`). Paths without `/` prefix are relative to root.

5. **Operation alignment with driver**: CLI commands map 1:1 to `pkg/driver/` methods. No CLI-only business logic.
