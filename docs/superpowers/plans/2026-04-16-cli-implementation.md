# 115driver CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Cobra-based CLI tool for 115 cloud storage with dual-mode output (human table + JSON for AI agents).

**Architecture:** Independent `cli/` directory parallel to `mcp/`, both consuming `pkg/driver/`. Auth via cookie flag/env/TOML config. Path resolution via existing `DirName2CID` API.

**Tech Stack:** Go 1.23, cobra, viper, tablewriter, cheggaaa/pb/v3, pkg/driver/

---

## File Map

| File | Responsibility |
|------|---------------|
| `cli/main.go` | Entry point, exit code handling |
| `cli/cmd/root.go` | Root command, global flags, PersistentPreRunE auth init |
| `cli/cmd/login.go` | QR code + cookie login, save to config |
| `cli/cmd/whoami.go` | Show current user info |
| `cli/cmd/ls.go` | List directory contents |
| `cli/cmd/stat.go` | File/directory details |
| `cli/cmd/mkdir.go` | Create directory |
| `cli/cmd/rename.go` | Rename file/directory |
| `cli/cmd/mv.go` | Move files into directory |
| `cli/cmd/cp.go` | Copy files into directory |
| `cli/cmd/rm.go` | Delete to recycle bin |
| `cli/cmd/upload.go` | Upload file |
| `cli/cmd/download.go` | Download file |
| `cli/cmd/search.go` | Search files |
| `cli/cmd/offline.go` | Offline download add/list/rm |
| `cli/internal/auth/auth.go` | Credential resolution + config file I/O |
| `cli/internal/output/printer.go` | Printer interface + envelope types |
| `cli/internal/output/table.go` | Human-readable table output |
| `cli/internal/output/json.go` | JSON output |
| `cli/internal/output/progress.go` | Progress bar wrapper |
| `cli/internal/resolver/resolver.go` | Path→ID resolution |

---

## Task 1: Project Scaffold + Dependencies

**Files:**
- Create: `cli/main.go`
- Create: `cli/cmd/root.go`
- Create: `cli/internal/auth/auth.go`
- Create: `cli/internal/output/printer.go`

- [ ] **Step 1: Install dependencies**

```bash
cd /Users/sheltonzhu/github/115driver
go get github.com/spf13/cobra@latest github.com/spf13/viper@latest github.com/olekukonez/tablewriter@latest github.com/cheggaaa/pb/v3@latest
go mod tidy
```

- [ ] **Step 2: Create `cli/internal/output/printer.go`**

```go
package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// Exit codes
const (
	ExitSuccess = 0
	ExitError   = 1
	ExitAuth    = 2
	ExitNotFound = 3
	ExitNetwork = 4
	ExitArgs    = 5
)

// Envelope is the unified JSON output wrapper.
type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Code    int         `json:"code"`
}

// Printer handles command output formatting.
type Printer struct {
	JSON bool
}

// NewPrinter creates a Printer based on json flag.
func NewPrinter(json bool) *Printer {
	return &Printer{JSON: json}
}

// PrintSuccess outputs success result.
func (p *Printer) PrintSuccess(data interface{}) {
	if p.JSON {
		p.printJSON(Envelope{Success: true, Data: data, Code: 0})
		return
	}
	// Human mode: data is printed by each command's specific handler
}

// PrintError outputs error result and returns exit code.
func (p *Printer) PrintError(msg string, code int) int {
	if p.JSON {
		p.printJSON(Envelope{Success: false, Error: msg, Code: code})
		return code
	}
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	return code
}

func (p *Printer) printJSON(env Envelope) {
	bytes, _ := json.MarshalIndent(env, "", "  ")
	fmt.Println(string(bytes))
}
```

- [ ] **Step 3: Create `cli/internal/auth/auth.go`**

```go
package auth

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SheltonZhu/115driver/pkg/driver"
	"github.com/spf13/viper"
)

const (
	DefaultConfigDir   = ".115driver"
	DefaultConfigFile  = "config.toml"
	DefaultProfile     = "main"
	EnvCookie          = "115DRIVER_COOKIE"
	EnvConfig          = "115DRIVER_CONFIG"
	EnvProfile         = "115DRIVER_PROFILE"
	EnvDebug           = "115DRIVER_DEBUG"
)

// Config represents the TOML config file structure.
type Config struct {
	DefaultProfile string                           `mapstructure:"default_profile"`
	Profiles       map[string]Profile                `mapstructure:"profiles"`
}

// Profile holds cookie for a named profile.
type Profile struct {
	Cookie string `mapstructure:"cookie"`
}

// ResolveCredential resolves credential from flag > env > config file.
// Returns error if no credential found.
func ResolveCredential(cookieFlag, configPath, profile string) (*driver.Credential, error) {
	// 1. --cookie flag
	if cookieFlag != "" {
		cr := &driver.Credential{}
		if err := cr.FromCookie(cookieFlag); err != nil {
			return nil, fmt.Errorf("invalid cookie: %w", err)
		}
		return cr, nil
	}

	// 2. Environment variable
	if envCookie := os.Getenv(EnvCookie); envCookie != "" {
		cr := &driver.Credential{}
		if err := cr.FromCookie(envCookie); err != nil {
			return nil, fmt.Errorf("invalid cookie from env: %w", err)
		}
		return cr, nil
	}

	// 3. Config file
	path := configPath
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, DefaultConfigDir, DefaultConfigFile)
	}

	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(*os.PathError); ok || os.IsNotExist(err) {
			return nil, fmt.Errorf("no credential found. Run '115driver login' to authenticate")
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if profile == "" {
		profile = v.GetString("default_profile")
		if profile == "" {
			profile = DefaultProfile
		}
	}

	cookieStr := v.GetString("profiles." + profile + ".cookie")
	if cookieStr == "" {
		return nil, fmt.Errorf("no credential found for profile '%s'. Run '115driver login'", profile)
	}

	cr := &driver.Credential{}
	if err := cr.FromCookie(cookieStr); err != nil {
		return nil, fmt.Errorf("invalid cookie in config: %w", err)
	}
	return cr, nil
}

// SaveCredential saves cookie to config file under given profile.
func SaveCredential(configPath, profile, cookie string) error {
	path := configPath
	if path == "" {
		home, _ := os.UserHomeDir()
		dir := filepath.Join(home, DefaultConfigDir)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}
		path = filepath.Join(dir, DefaultConfigFile)
	}

	v := viper.New()
	v.SetConfigFile(path)
	_ = v.ReadInConfig()

	if profile == "" {
		profile = DefaultProfile
	}

	v.Set("default_profile", profile)
	v.Set("profiles."+profile+".cookie", cookie)

	return v.WriteConfig()
}
```

- [ ] **Step 4: Create `cli/cmd/root.go`**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/SheltonZhu/115driver/cli/internal/auth"
	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/SheltonZhu/115driver/pkg/driver"
	"github.com/spf13/cobra"
)

var (
	cookieFlag string
	configPath string
	profile    string
	jsonOutput bool
	debugMode  bool
)

// client is the authenticated driver client, set during PersistentPreRunE.
var client *driver.Pan115Client

// printer is the output printer, set during PersistentPreRunE.
var printer *output.Printer

var rootCmd = &cobra.Command{
	Use:   "115driver",
	Short: "CLI tool for 115 cloud storage",
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		printer = output.NewPrinter(jsonOutput)

		// Skip auth for login command
		if cmd.Name() == "login" {
			return nil
		}

		cr, err := auth.ResolveCredential(cookieFlag, configPath, profile)
		if err != nil {
			return &exitError{code: output.ExitAuth, msg: err.Error()}
		}

		opts := []driver.Option{driver.UA(driver.UA115Browser)}
		if debugMode {
			opts = append(opts, driver.WithDebug())
		}
		client = driver.New(opts...).ImportCredential(cr)

		if err := client.LoginCheck(); err != nil {
			return &exitError{code: output.ExitAuth, msg: fmt.Sprintf("Authentication failed: %v\nRun '115driver login' to re-authenticate.", err)}
		}
		return nil
	},
}

type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string {
	return e.msg
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cookieFlag, "cookie", "", "Cookie string (or set 115DRIVER_COOKIE env)")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Config file path (default ~/.115driver/config.toml)")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "Config profile name (default 'main')")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format (for AI agents)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug mode")
}

// Execute runs the root command.
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		if ee, ok := err.(*exitError); ok {
			return printer.PrintError(ee.msg, ee.code)
		}
		return printer.PrintError(err.Error(), output.ExitError)
	}
	return output.ExitSuccess
}
```

- [ ] **Step 5: Create `cli/main.go`**

```go
package main

import (
	"os"

	"github.com/SheltonZhu/115driver/cli/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
```

- [ ] **Step 6: Build and verify**

```bash
go build -o 115driver ./cli/
./115driver --help
```

Expected: help text showing available subcommands (none yet) and global flags.

- [ ] **Step 7: Commit**

```bash
git add cli/ go.mod go.sum
git commit -m "feat(cli): scaffold CLI with cobra, auth, and output system"
```

---

## Task 2: Output System — Table Rendering

**Files:**
- Create: `cli/internal/output/table.go`
- Create: `cli/internal/output/json.go`

- [ ] **Step 1: Create `cli/internal/output/json.go`**

```go
package output

import (
	"time"
)

// JSONFile represents a file in JSON output.
type JSONFile struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	IsDir      bool   `json:"is_dir"`
	UpdateTime string `json:"update_time,omitempty"`
	FileID     string `json:"file_id"`
	PickCode   string `json:"pick_code,omitempty"`
	Sha1       string `json:"sha1,omitempty"`
}

// FileToJSON converts a driver.File to JSONFile.
func FileToJSON(f interface {
	GetName() string
	GetSize() int64
	IsDir() bool
	GetID() string
	ModTime() time.Time
}) JSONFile {
	return JSONFile{
		Name:       f.GetName(),
		Size:       f.GetSize(),
		IsDir:      f.IsDir(),
		UpdateTime: f.ModTime().Format(time.RFC3339),
		FileID:     f.GetID(),
	}
}

// JSONStat represents stat output.
type JSONStat struct {
	Name       string      `json:"name"`
	Size       int64       `json:"size"`
	IsDir      bool        `json:"is_dir"`
	FileID     string      `json:"file_id"`
	Sha1       string      `json:"sha1,omitempty"`
	PickCode   string      `json:"pick_code,omitempty"`
	CreateTime string      `json:"create_time"`
	UpdateTime string      `json:"update_time"`
	Parents    []JSONDir   `json:"parents,omitempty"`
	FileCount  int         `json:"file_count,omitempty"`
	DirCount   int         `json:"dir_count,omitempty"`
}

// JSONDir represents a directory in path.
type JSONDir struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
```

- [ ] **Step 2: Create `cli/internal/output/table.go`**

```go
package output

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/olekukonez/tablewriter"
)

// FormatFileSize converts bytes to human-readable string.
func FormatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/float64(TB))
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// FormatTime formats time for display.
func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

// PrintFileTable prints files in table format for human mode.
func (p *Printer) PrintFileTable(path string, files []JSONFile) {
	if p.JSON {
		p.PrintSuccess(map[string]interface{}{
			"path":  path,
			"files": files,
		})
		return
	}

	fmt.Printf("Path: %s\n\n", path)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Size", "Type", "Modified"})
	table.SetBorder(false)
	table.SetAutoWrapText(false)

	for _, f := range files {
		typ := "file"
		if f.IsDir {
			typ = "dir"
		}
		name := f.Name
		if f.IsDir {
			name = name + "/"
		}
		table.Append([]string{
			name,
			FormatFileSize(f.Size),
			typ,
			f.UpdateTime,
		})
	}
	table.Render()
}

// PrintFileList prints a simple name-only list (non -l mode).
func (p *Printer) PrintFileList(path string, files []JSONFile) {
	if p.JSON {
		p.PrintSuccess(map[string]interface{}{
			"path":  path,
			"files": files,
		})
		return
	}

	for _, f := range files {
		if f.IsDir {
			fmt.Printf("%s/\n", f.Name)
		} else {
			fmt.Println(f.Name)
		}
	}
}

// PrintStatTable prints file details for stat command.
func (p *Printer) PrintStatTable(stat JSONStat) {
	if p.JSON {
		p.PrintSuccess(stat)
		return
	}

	fmt.Printf("  Name:   %s\n", stat.Name)
	fmt.Printf("  Type:   %s\n", map[bool]string{true: "Directory", false: "File"}[stat.IsDir])
	fmt.Printf("  Size:   %s\n", FormatFileSize(stat.Size))
	if stat.Sha1 != "" {
		fmt.Printf("  SHA1:   %s\n", stat.Sha1)
	}
	if stat.PickCode != "" {
		fmt.Printf("  Pick:   %s\n", stat.PickCode)
	}
	fmt.Printf("  ID:     %s\n", stat.FileID)
	fmt.Printf("  Created:  %s\n", stat.CreateTime)
	fmt.Printf("  Modified: %s\n", stat.UpdateTime)
	if stat.IsDir {
		fmt.Printf("  Files:  %d\n", stat.FileCount)
		fmt.Printf("  Dirs:   %d\n", stat.DirCount)
	}
	if len(stat.Parents) > 0 {
		var parts []string
		for _, p := range stat.Parents {
			parts = append(parts, p.Name)
		}
		fmt.Printf("  Path:   /%s\n", strings.Join(parts, "/"))
	}
}

// PrintSuccessMsg prints a simple success message for human mode.
func (p *Printer) PrintSuccessMsg(data map[string]interface{}) {
	if p.JSON {
		p.PrintSuccess(data)
		return
	}
	// Simple key-value for operation results
	if msg, ok := data["message"]; ok {
		fmt.Println(msg)
	}
}
```

- [ ] **Step 3: Build and verify**

```bash
go build -o 115driver ./cli/
```

Expected: compiles without errors.

- [ ] **Step 4: Commit**

```bash
git add cli/internal/output/
git commit -m "feat(cli): add table and JSON output formatters"
```

---

## Task 3: Path Resolver

**Files:**
- Create: `cli/internal/resolver/resolver.go`

- [ ] **Step 1: Create `cli/internal/resolver/resolver.go`**

```go
package resolver

import (
	"fmt"
	"path"
	"strings"

	"github.com/SheltonZhu/115driver/pkg/driver"
)

const RootID = "0"

// ResolveDir resolves a remote path to a directory ID.
// Uses driver.DirName2CID for direct lookup.
func ResolveDir(client *driver.Pan115Client, remotePath string) (string, error) {
	if remotePath == "" || remotePath == "/" {
		return RootID, nil
	}

	// TODO: add LRU cache for path resolution
	cleaned := strings.TrimPrefix(remotePath, "/")
	cleaned = strings.TrimSuffix(cleaned, "/")

	if cleaned == "" {
		return RootID, nil
	}

	resp, err := client.DirName2CID(cleaned)
	if err != nil {
		return "", fmt.Errorf("directory not found: %s (%w)", remotePath, err)
	}
	return string(resp.CategoryID), nil
}

// ResolveFile resolves a remote file path to a file ID.
// It resolves the parent directory first, then lists and matches by name.
func ResolveFile(client *driver.Pan115Client, remotePath string) (string, error) {
	cleaned := strings.TrimPrefix(remotePath, "/")
	cleaned = strings.TrimSuffix(cleaned, "/")

	dir := path.Dir(cleaned)
	fileName := path.Base(cleaned)

	dirID, err := ResolveDir(client, "/"+dir)
	if err != nil {
		return "", err
	}

	files, err := client.List(dirID)
	if err != nil {
		return "", fmt.Errorf("failed to list directory: %w", err)
	}

	for _, f := range *files {
		if f.Name == fileName && !f.IsDirectory {
			return f.FileID, nil
		}
	}
	return "", fmt.Errorf("file not found: %s", remotePath)
}

// ResolvePath resolves a remote path to a file or directory ID.
// Returns (fileID, isDir, error).
// Tries directory resolution first, falls back to file resolution.
func ResolvePath(client *driver.Pan115Client, remotePath string) (string, bool, error) {
	if remotePath == "" || remotePath == "/" {
		return RootID, true, nil
	}

	// Try as directory first
	dirID, err := ResolveDir(client, remotePath)
	if err == nil && dirID != "" && dirID != "0" || (remotePath == "/" || remotePath == "") {
		return dirID, true, nil
	}
	if remotePath == "/" {
		return RootID, true, nil
	}

	// Try as file
	fileID, err := ResolveFile(client, remotePath)
	if err != nil {
		// Both failed — try one more time as directory (might be root subdir)
		dirID2, err2 := ResolveDir(client, remotePath)
		if err2 == nil {
			return dirID2, true, nil
		}
		return "", false, fmt.Errorf("path not found: %s", remotePath)
	}
	return fileID, false, nil
}
```

- [ ] **Step 2: Build and verify**

```bash
go build -o 115driver ./cli/
```

Expected: compiles without errors.

- [ ] **Step 3: Commit**

```bash
git add cli/internal/resolver/
git commit -m "feat(cli): add path resolver using DirName2CID + List"
```

---

## Task 4: Login + Whoami Commands

**Files:**
- Create: `cli/cmd/login.go`
- Create: `cli/cmd/whoami.go`
- Modify: `cli/cmd/root.go` (add subcommand registration)

- [ ] **Step 1: Create `cli/cmd/login.go`**

```go
package cmd

import (
	"fmt"
tt"context"
	"time"

	"github.com/SheltonZhu/115driver/cli/internal/auth"
	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/SheltonZhu/115driver/pkg/driver"
	"github.com/spf13/cobra"
)

var loginCookie string

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with 115 cloud storage",
	RunE: func(cmd *cobra.Command, args []string) error {
		if loginCookie != "" {
			return loginWithCookie()
		}
		return loginWithQRCode()
	},
}

func init() {
	loginCmd.Flags().StringVar(&loginCookie, "cookie", "", "Cookie string for authentication")
	rootCmd.AddCommand(loginCmd)
}

func loginWithCookie() error {
	p := output.NewPrinter(jsonOutput)

	cr := &driver.Credential{}
	if err := cr.FromCookie(loginCookie); err != nil {
		return &exitError{code: output.ExitArgs, msg: fmt.Sprintf("Invalid cookie: %v", err)}
	}

	c := driver.New(driver.UA(driver.UA115Browser)).ImportCredential(cr)
	if err := c.LoginCheck(); err != nil {
		return &exitError{code: output.ExitAuth, msg: fmt.Sprintf("Cookie validation failed: %v", err)}
	}

	if err := auth.SaveCredential(configPath, profile, loginCookie); err != nil {
		return &exitError{code: output.ExitError, msg: fmt.Sprintf("Failed to save config: %v", err)}
	}

	p.PrintSuccess(map[string]interface{}{
		"profile":      auth.DefaultProfile,
		"cookie_saved": true,
	})
	if !jsonOutput {
		fmt.Println("Login successful. Cookie saved to config.")
	}
	return nil
}

func loginWithQRCode() error {
	p := output.NewPrinter(jsonOutput)

	c := driver.New(driver.UA(driver.UA115Browser))
	session, err := c.QRCodeStart()
	if err != nil {
		return &exitError{code: output.ExitError, msg: fmt.Sprintf("Failed to start QR login: %v", err)}
	}

	if jsonOutput {
		fmt.Printf(`{"success":true,"data":{"qr_url":"https://qrcodeapi.115.com/api/1.0/mac/1.0/qrcode?uid=%s","message":"Scan QR code with 115 app"}}`+"\n", session.UID)
	} else {
		fmt.Println("Scan the QR code with 115 app to login:")
		fmt.Printf("URL: https://qrcodeapi.115.com/api/1.0/mac/1.0/qrcode?uid=%s\n\n", session.UID)
	}

	// Poll for QR code status with context for graceful Ctrl+C
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return &exitError{code: output.ExitError, msg: "QR code login timed out."}
		default:
		}

		time.Sleep(3 * time.Second)
		status, err := c.QRCodeStatus(session)
		if err != nil {
			continue
		}

		if status.IsExpired() {
			return &exitError{code: output.ExitError, msg: "QR code expired. Please try again."}
		}
		if status.IsCanceled() {
			return &exitError{code: output.ExitError, msg: "QR code login canceled."}
		}
		if status.IsAllowed() {
			cred, err := c.QRCodeLogin(session)
			if err != nil {
				return &exitError{code: output.ExitAuth, msg: fmt.Sprintf("Login failed: %v", err)}
			}

			cookieStr := cred.Cookie()
			if err := auth.SaveCredential(configPath, profile, cookieStr); err != nil {
				return &exitError{code: output.ExitError, msg: fmt.Sprintf("Failed to save config: %v", err)}
			}

			if jsonOutput {
				p.PrintSuccess(map[string]interface{}{
					"profile":      auth.DefaultProfile,
					"cookie_saved": true,
				})
			} else {
				fmt.Println("Login successful. Cookie saved to config.")
			}
			return nil
		}
		if !jsonOutput {
			fmt.Print(".")
		}
	}
}
```

- [ ] **Step 2: Create `cli/cmd/whoami.go`**

```go
package cmd

import (
	"fmt"

	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current authenticated user info",
	RunE: func(cmd *cobra.Command, args []string) error {
		userInfo, err := client.GetUser()
		if err != nil {
			return &exitError{code: output.ExitAuth, msg: fmt.Sprintf("Failed to get user info: %v", err)}
		}

		if jsonOutput {
			printer.PrintSuccess(map[string]interface{}{
				"user_id":  userInfo.UserID,
				"username": userInfo.UserName,
				"vip":      userInfo.Vip,
				"expire":   userInfo.Expire,
			})
		} else {
			fmt.Printf("User ID:  %d\n", userInfo.UserID)
			fmt.Printf("Username: %s\n", userInfo.UserName)
			if userInfo.Vip > 0 {
				fmt.Printf("VIP:      %d (expires: %d)\n", userInfo.Vip, userInfo.Expire)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
```

- [ ] **Step 3: Build and test**

```bash
go build -o 115driver ./cli/
./115driver login --help
./115driver whoami --help
```

Expected: both commands show help text.

- [ ] **Step 4: Commit**

```bash
git add cli/cmd/login.go cli/cmd/whoami.go
git commit -m "feat(cli): add login and whoami commands"
```

---

## Task 5: ls Command

**Files:**
- Create: `cli/cmd/ls.go`

- [ ] **Step 1: Create `cli/cmd/ls.go`**

```go
package cmd

import (
	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/SheltonZhu/115driver/cli/internal/resolver"
	"github.com/spf13/cobra"
)

var lsLong bool

var lsCmd = &cobra.Command{
	Use:   "ls [remote_path]",
	Short: "List directory contents",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		remotePath := "/"
		if len(args) > 0 {
			remotePath = args[0]
		}

		dirID, err := resolver.ResolveDir(client, remotePath)
		if err != nil {
			return &exitError{code: output.ExitNotFound, msg: err.Error()}
		}

		files, err := client.List(dirID)
		if err != nil {
			return &exitError{code: output.ExitError, msg: err.Error()}
		}

		jsonFiles := make([]output.JSONFile, 0, len(*files))
		for _, f := range *files {
			jsonFiles = append(jsonFiles, output.FileToJSON(&f))
		}

		if lsLong {
			printer.PrintFileTable(remotePath, jsonFiles)
		} else {
			printer.PrintFileList(remotePath, jsonFiles)
		}
		return nil
	},
}

func init() {
	lsCmd.Flags().BoolVarP(&lsLong, "long", "l", false, "Show detailed listing")
	rootCmd.AddCommand(lsCmd)
}
```

- [ ] **Step 2: Build and verify**

```bash
go build -o 115driver ./cli/
./115driver ls --help
```

Expected: help shows `-l` flag.

- [ ] **Step 3: Commit**

```bash
git add cli/cmd/ls.go
git commit -m "feat(cli): add ls command with table and list output"
```

---

## Task 6: stat Command

**Files:**
- Create: `cli/cmd/stat.go`

- [ ] **Step 1: Create `cli/cmd/stat.go`**

```go
package cmd

import (
	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/SheltonZhu/115driver/cli/internal/resolver"
	"github.com/spf13/cobra"
)

var statCmd = &cobra.Command{
	Use:   "stat <remote_path>",
	Short: "Show file or directory details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		remotePath := args[0]
		fileID, _, err := resolver.ResolvePath(client, remotePath)
		if err != nil {
			return &exitError{code: output.ExitNotFound, msg: err.Error()}
		}

		statInfo, err := client.Stat(fileID)
		if err != nil {
			return &exitError{code: output.ExitError, msg: err.Error()}
		}

		jsonStat := output.JSONStat{
			Name:       statInfo.Name,
			IsDir:      statInfo.IsDirectory,
			Sha1:       statInfo.Sha1,
				FileID:     fileID,
			PickCode:   statInfo.PickCode,
			CreateTime: statInfo.CreateTime.Format("2006-01-02 15:04:05"),
			UpdateTime: statInfo.UpdateTime.Format("2006-01-02 15:04:05"),
			FileCount:  statInfo.FileCount,
			DirCount:   statInfo.DirCount,
			Parents:    make([]output.JSONDir, 0, len(statInfo.Parents)),
		}

		// Get size from GetFile if it's a file
		if !statInfo.IsDirectory {
			f, err := client.GetFile(fileID)
			if err == nil {
				jsonStat.Size = f.Size
			}
		}

		for _, p := range statInfo.Parents {
			jsonStat.Parents = append(jsonStat.Parents, output.JSONDir{ID: p.ID, Name: p.Name})
		}

		printer.PrintStatTable(jsonStat)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statCmd)
}
```

- [ ] **Step 2: Build and verify**

```bash
go build -o 115driver ./cli/
```

Expected: compiles.

- [ ] **Step 3: Commit**

```bash
git add cli/cmd/stat.go
git commit -m "feat(cli): add stat command"
```

---

## Task 7: mkdir Command

**Files:**
- Create: `cli/cmd/mkdir.go`

- [ ] **Step 1: Create `cli/cmd/mkdir.go`**

```go
package cmd

import (
	"fmt"
	"path"
	"strings"

	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/SheltonZhu/115driver/cli/internal/resolver"
	"github.com/SheltonZhu/115driver/pkg/driver"
	"github.com/spf13/cobra"
)

var mkdirParents bool

var mkdirCmd = &cobra.Command{
	Use:   "mkdir [-p] <remote_path>",
	Short: "Create a directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		remotePath := args[0]
		remotePath = strings.TrimSuffix(remotePath, "/")
		dirName := path.Base(remotePath)
		parentPath := path.Dir(remotePath)

		if mkdirParents {
			return mkdirP(parentPath, dirName, remotePath)
		}

		parentID, err := resolver.ResolveDir(client, parentPath)
		if err != nil {
			return &exitError{code: output.ExitNotFound, msg: fmt.Sprintf("Parent directory not found: %s", parentPath)}
		}

		dirID, err := client.Mkdir(parentID, dirName)
		if err != nil {
			return &exitError{code: output.ExitError, msg: err.Error()}
		}

		printer.PrintSuccess(map[string]interface{}{
			"name":   dirName,
			"dir_id": dirID,
			"path":   remotePath,
		})
		if !jsonOutput {
			fmt.Printf("Created directory: %s (ID: %s)\n", remotePath, dirID)
		}
		return nil
	},
}


```go
func mkdirP(parentPath, dirName, fullPath string) error {
	parts := strings.Split(strings.Trim(parentPath+"/"+dirName, "/"), "/")
	currentID := resolver.RootID
	createdPath := ""

	for _, part := range parts {
		if part == "" {
			continue
		}
		createdPath += "/" + part

		existingID, err := resolver.ResolveDir(client, createdPath)
		if err == nil && existingID != "" {
			currentID = existingID
			continue
		}

		newID, err := client.Mkdir(currentID, part)
		if err != nil {
			if isExistErr(err) {
				existingID, _ := resolver.ResolveDir(client, createdPath)
				if existingID != "" {
					currentID = existingID
					continue
				}
			}
			return &exitError{code: output.ExitError, msg: err.Error()}
		}
		currentID = newID
	}

	printer.PrintSuccess(map[string]interface{}{
		"name":   dirName,
		"dir_id": currentID,
		"path":   fullPath,
	})
	if !jsonOutput {
		fmt.Printf("Created directory: %s (ID: %s)\n", fullPath, currentID)
	}
	return nil
}
```

- [ ] **Step 2: Build and verify**

```bash
go build -o 115driver ./cli/
```

- [ ] **Step 3: Commit**

```bash
git add cli/cmd/mkdir.go
git commit -m "feat(cli): add mkdir command with -p flag"
```

---

## Task 8: rename, mv, cp Commands

**Files:**
- Create: `cli/cmd/rename.go`
- Create: `cli/cmd/mv.go`
- Create: `cli/cmd/cp.go`

- [ ] **Step 1: Create `cli/cmd/rename.go`**

```go
package cmd

import (
	"fmt"
	"path"

	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/SheltonZhu/115driver/cli/internal/resolver"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename <remote_path> <new_name>",
	Short: "Rename a file or directory",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		remotePath := args[0]
		newName := args[1]

		fileID, _, err := resolver.ResolvePath(client, remotePath)
		if err != nil {
			return &exitError{code: output.ExitNotFound, msg: err.Error()}
		}

		if err := client.Rename(fileID, newName); err != nil {
			return &exitError{code: output.ExitError, msg: err.Error()}
		}

		printer.PrintSuccess(map[string]interface{}{
			"old_name": path.Base(remotePath),
			"new_name": newName,
			"file_id":  fileID,
		})
		if !jsonOutput {
			fmt.Printf("Renamed to: %s\n", newName)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
```

- [ ] **Step 2: Create `cli/cmd/mv.go`**

```go
package cmd

import (
	"fmt"

	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/SheltonZhu/115driver/cli/internal/resolver"
	"github.com/spf13/cobra"
)

var mvCmd = &cobra.Command{
	Use:   "mv <source_path> <destination_dir>",
	Short: "Move files into a destination directory",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		srcPath := args[0]
		dstDir := args[1]

		fileID, _, err := resolver.ResolvePath(client, srcPath)
		if err != nil {
			return &exitError{code: output.ExitNotFound, msg: err.Error()}
		}

		dirID, err := resolver.ResolveDir(client, dstDir)
		if err != nil {
			return &exitError{code: output.ExitNotFound, msg: fmt.Sprintf("Destination directory not found: %s", dstDir)}
		}

		if err := client.Move(dirID, fileID); err != nil {
			return &exitError{code: output.ExitError, msg: err.Error()}
		}

		printer.PrintSuccess(map[string]interface{}{
			"source":          srcPath,
			"destination_dir": dstDir,
			"file_ids":        []string{fileID},
		})
		if !jsonOutput {
			fmt.Printf("Moved %s -> %s\n", srcPath, dstDir)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mvCmd)
}
```

- [ ] **Step 3: Create `cli/cmd/cp.go`**

```go
package cmd

import (
	"fmt"

	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/SheltonZhu/115driver/cli/internal/resolver"
	"github.com/spf13/cobra"
)

var cpCmd = &cobra.Command{
	Use:   "cp <source_path> <destination_dir>",
	Short: "Copy files into a destination directory",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		srcPath := args[0]
		dstDir := args[1]

		fileID, _, err := resolver.ResolvePath(client, srcPath)
		if err != nil {
			return &exitError{code: output.ExitNotFound, msg: err.Error()}
		}

		dirID, err := resolver.ResolveDir(client, dstDir)
		if err != nil {
			return &exitError{code: output.ExitNotFound, msg: fmt.Sprintf("Destination directory not found: %s", dstDir)}
		}

		if err := client.Copy(dirID, fileID); err != nil {
			return &exitError{code: output.ExitError, msg: err.Error()}
		}

		printer.PrintSuccess(map[string]interface{}{
			"source":          srcPath,
			"destination_dir": dstDir,
			"file_ids":        []string{fileID},
		})
		if !jsonOutput {
			fmt.Printf("Copied %s -> %s\n", srcPath, dstDir)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cpCmd)
}
```

- [ ] **Step 4: Build and verify**

```bash
go build -o 115driver ./cli/
```

- [ ] **Step 5: Commit**

```bash
git add cli/cmd/rename.go cli/cmd/mv.go cli/cmd/cp.go
git commit -m "feat(cli): add rename, mv, and cp commands"
```

---

## Task 9: rm Command

**Files:**
- Create: `cli/cmd/rm.go`

- [ ] **Step 1: Create `cli/cmd/rm.go`**

```go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/SheltonZhu/115driver/cli/internal/resolver"
	"github.com/spf13/cobra"
)

var rmForce bool

var rmCmd = &cobra.Command{
	Use:   "rm <remote_path>",
	Short: "Delete file or directory (moves to recycle bin)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		remotePath := args[0]

		fileID, isDir, err := resolver.ResolvePath(client, remotePath)
		if err != nil {
			return &exitError{code: output.ExitNotFound, msg: err.Error()}
		}

		// Confirmation for directories in human mode
		if isDir && !jsonOutput {
			fmt.Printf("Delete directory %s and all its contents? [y/N] ", remotePath)
			reader := bufio.NewReader(os.Stdin)
			resp, _ := reader.ReadString('\n')
			resp = strings.TrimSpace(strings.ToLower(resp))
			if resp != "y" && resp != "yes" {
				fmt.Println("Canceled.")
				return nil
			}
		}

		if err := client.Delete(fileID); err != nil {
			return &exitError{code: output.ExitError, msg: err.Error()}
		}

		printer.PrintSuccess(map[string]interface{}{
			"deleted":  []string{remotePath},
			"file_ids": []string{fileID},
		})
		if !jsonOutput {
			fmt.Printf("Deleted: %s\n", remotePath)
		}
		return nil
	},
}

func init() {
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Reserved for future permanent delete")
	rootCmd.AddCommand(rmCmd)
}
```

- [ ] **Step 2: Build and verify**

```bash
go build -o 115driver ./cli/
```

- [ ] **Step 3: Commit**

```bash
git add cli/cmd/rm.go
git commit -m "feat(cli): add rm command with confirmation for directories"
```

---

## Task 10: search Command

**Files:**
- Create: `cli/cmd/search.go`

- [ ] **Step 1: Create `cli/cmd/search.go`**

```go
package cmd

import (
	"fmt"

	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/SheltonZhu/115driver/pkg/driver"
	"github.com/spf13/cobra"
)

var (
	searchType  string
	searchSort  string
	searchLimit int
)

// typeMap maps CLI type names to driver SearchOption.Type values.
var typeMap = map[string]int{
	"folder":   1,
	"document": 2,
	"image":    3,
	"video":    4,
	"audio":    5,
	"archive":  6,
}

var searchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "Search for files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		keyword := args[0]

		opts := &driver.SearchOption{
			SearchValue: keyword,
			Limit:       searchLimit,
		}

		if t, ok := typeMap[searchType]; ok {
			opts.Type = t
		}
		if searchSort != "" {
			opts.Order = searchSort
		}

		result, err := client.Search(opts)
		if err != nil {
			return &exitError{code: output.ExitError, msg: err.Error()}
		}

		jsonFiles := make([]output.JSONFile, 0, len(result.Files))
		for i := range result.Files {
			jsonFiles = append(jsonFiles, output.FileToJSON(&result.Files[i]))
		}

		if jsonOutput {
			printer.PrintSuccess(map[string]interface{}{
				"keyword": keyword,
				"count":   result.Count,
				"files":   jsonFiles,
			})
		} else {
			fmt.Printf("Found %d results for '%s':\n\n", result.Count, keyword)
			printer.PrintFileTable("", jsonFiles)
		}
		return nil
	},
}

func init() {
	searchCmd.Flags().StringVarP(&searchType, "type", "t", "", "Filter by type: folder, document, image, video, audio, archive")
	searchCmd.Flags().StringVar(&searchSort, "sort", "", "Sort field (e.g. file_name, file_size, user_ptime)")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 30, "Max results to return")
	rootCmd.AddCommand(searchCmd)
}
```

- [ ] **Step 2: Build and verify**

```bash
go build -o 115driver ./cli/
```

- [ ] **Step 3: Commit**

```bash
git add cli/cmd/search.go
git commit -m "feat(cli): add search command with type filter and sort"
```

---

## Task 11: upload + download Commands

**Files:**
- Create: `cli/internal/output/progress.go`
- Create: `cli/cmd/upload.go`
- Create: `cli/cmd/download.go`

- [ ] **Step 1: Create `cli/internal/output/progress.go`**

```go
package output

import (
	"fmt"
	"io"
	"os"

	"github.com/cheggaaa/pb/v3"
)

// CreateProgressBar creates a progress bar for the given size.
// Returns nil bar if not a terminal (e.g., piped output).
func CreateProgressBar(total int64) *pb.ProgressBar {
	if !isTerminal() {
		return nil
	}
	bar := pb.Start64(total)
	bar.SetTemplateString(`{{counters . }} {{bar . }} {{percent . }} {{speed . }}`)
	return bar
}

func isTerminal() bool {
	fi, _ := os.Stdout.Stat()
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// TrackProgress wraps a reader with a progress bar.
func TrackProgress(r io.Reader, total int64) io.Reader {
	bar := CreateProgressBar(total)
	if bar == nil {
		return r
	}
	return bar.NewProxyReader(r)
}

// FinishProgress finishes a progress bar if it exists.
func FinishProgress(bar *pb.ProgressBar) {
	if bar != nil {
		bar.Finish()
	}
}

// PrintProgress prints a simple progress message for non-terminal.
func PrintProgress(done, total int64) {
	fmt.Printf("\rProgress: %s / %s", FormatFileSize(done), FormatFileSize(total))
}
```

- [ ] **Step 2: Create `cli/cmd/upload.go`**

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/SheltonZhu/115driver/cli/internal/resolver"
	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:   "upload <local_path> <remote_dir>",
	Short: "Upload a file to remote directory",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		localPath := args[0]
		remoteDir := args[1]

		dirID, err := resolver.ResolveDir(client, remoteDir)
		if err != nil {
			return &exitError{code: output.ExitNotFound, msg: fmt.Sprintf("Remote directory not found: %s", remoteDir)}
		}

		f, err := os.Open(localPath)
		if err != nil {
			return &exitError{code: output.ExitArgs, msg: fmt.Sprintf("Cannot open local file: %v", err)}
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			return &exitError{code: output.ExitError, msg: err.Error()}
		}

		if stat.IsDir() {
			return &exitError{code: output.ExitArgs, msg: "Directory upload is not supported. Upload individual files."}
		}

		fileName := filepath.Base(localPath)

		if !jsonOutput {
			fmt.Printf("Uploading %s (%s)...\n", fileName, output.FormatFileSize(stat.Size()))
		}

		err = client.RapidUploadOrByOSS(dirID, fileName, stat.Size(), f)
		if err != nil {
			return &exitError{code: output.ExitError, msg: fmt.Sprintf("Upload failed: %v", err)}
		}

		printer.PrintSuccess(map[string]interface{}{
			"local_path": localPath,
			"remote_dir": remoteDir,
			"size":       stat.Size(),
		})
		if !jsonOutput {
			fmt.Printf("Upload complete: %s -> %s\n", fileName, remoteDir)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}
```

- [ ] **Step 3: Create `cli/cmd/download.go`**

```go
package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/SheltonZhu/115driver/cli/internal/resolver"
	"github.com/SheltonZhu/115driver/pkg/driver"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download <remote_path> <local_dir>",
	Short: "Download a file from remote to local directory",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		remotePath := args[0]
		localDir := args[1]

		// Resolve to get pickCode
		fileID, _, err := resolver.ResolvePath(client, remotePath)
		if err != nil {
			return &exitError{code: output.ExitNotFound, msg: err.Error()}
		}

		// Get file info for name
		fileInfo, err := client.GetFile(fileID)
		if err != nil {
			return &exitError{code: output.ExitError, msg: err.Error()}
		}
		if fileInfo.IsDirectory {
			return &exitError{code: output.ExitArgs, msg: "Cannot download a directory."}
		}

		dlInfo, err := client.Download(fileInfo.PickCode)
		if err != nil {
			return &exitError{code: output.ExitError, msg: fmt.Sprintf("Failed to get download URL: %v", err)}
		}

		localPath := filepath.Join(localDir, dlInfo.FileName)

		if !jsonOutput {
			fmt.Printf("Downloading %s (%s)...\n", dlInfo.FileName, output.FormatFileSize(int64(dlInfo.FileSize)))
		}

		if err := downloadFile(dlInfo, localPath); err != nil {
			return &exitError{code: output.ExitError, msg: fmt.Sprintf("Download failed: %v", err)}
		}

		printer.PrintSuccess(map[string]interface{}{
			"remote_path": remotePath,
			"local_path":  localPath,
			"size":        int64(dlInfo.FileSize),
		})
		if !jsonOutput {
			fmt.Printf("Download complete: %s\n", localPath)
		}
		return nil
	},
}

func downloadFile(dlInfo *driver.DownloadInfo, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}

	reader, err := dlInfo.Get()
	if err != nil {
		return fmt.Errorf("get download stream: %w", err)
	}

	out, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, reader)
	return err
}

func init() {
	rootCmd.AddCommand(downloadCmd)
}
```

- [ ] **Step 4: Build and verify**

```bash
go build -o 115driver ./cli/
```

- [ ] **Step 5: Commit**

```bash
git add cli/internal/output/progress.go cli/cmd/upload.go cli/cmd/download.go
git commit -m "feat(cli): add upload and download commands with progress"
```

---

## Task 12: offline Command (add/list/rm)

**Files:**
- Create: `cli/cmd/offline.go`

- [ ] **Step 1: Create `cli/cmd/offline.go`**

```go
package cmd

import (
	"fmt"

	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/SheltonZhu/115driver/cli/internal/resolver"
	"github.com/SheltonZhu/115driver/pkg/driver"
	"github.com/olekukonez/tablewriter"
	"github.com/spf13/cobra"
	"os"
)

var offlineSaveDir string

var offlineCmd = &cobra.Command{
	Use:   "offline",
	Short: "Manage offline downloads",
}

var offlineAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add an offline download task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]

		saveDirID := resolver.RootID
		if offlineSaveDir != "" {
			id, err := resolver.ResolveDir(client, offlineSaveDir)
			if err != nil {
				return &exitError{code: output.ExitNotFound, msg: fmt.Sprintf("Save directory not found: %s", offlineSaveDir)}
			}
			saveDirID = id
		}

		hashes, err := client.AddOfflineTaskURIs([]string{url}, saveDirID)
		if err != nil {
			return &exitError{code: output.ExitError, msg: err.Error()}
		}

		printer.PrintSuccess(map[string]interface{}{
			"url":       url,
			"hashes":    hashes,
			"save_dir":  offlineSaveDir,
		})
		if !jsonOutput {
			fmt.Printf("Offline task added: %s\n", url)
		}
		return nil
	},
}

var offlineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List offline download tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := client.ListOfflineTask(1)
		if err != nil {
			return &exitError{code: output.ExitError, msg: err.Error()}
		}

		if jsonOutput {
			tasks := make([]map[string]interface{}, 0, len(result.Tasks))
			for _, t := range result.Tasks {
				tasks = append(tasks, map[string]interface{}{
					"name":    t.Name,
					"hash":    t.InfoHash,
					"status":  t.GetStatus(),
					"percent": t.Percent,
					"size":    t.Size,
				})
			}
			printer.PrintSuccess(map[string]interface{}{
				"total": result.Total,
				"tasks": tasks,
			})
		} else {
			fmt.Printf("Offline tasks (%d total):\n\n", result.Total)
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Name", "Status", "Progress", "Size"})
			table.SetBorder(false)
			for _, t := range result.Tasks {
				table.Append([]string{
					t.Name,
					t.GetStatus(),
					fmt.Sprintf("%.1f%%", t.Percent),
					output.FormatFileSize(t.Size),
				})
			}
			table.Render()
		}
		return nil
	},
}

var offlineRmCmd = &cobra.Command{
	Use:   "rm <hash>",
	Short: "Remove an offline download task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hash := args[0]

		if err := client.DeleteOfflineTasks([]string{hash}, false); err != nil {
			return &exitError{code: output.ExitError, msg: err.Error()}
		}

		printer.PrintSuccess(map[string]interface{}{
			"deleted_hashes": []string{hash},
		})
		if !jsonOutput {
			fmt.Printf("Removed offline task: %s\n", hash)
		}
		return nil
	},
}

func init() {
	offlineAddCmd.Flags().StringVarP(&offlineSaveDir, "dir", "d", "", "Remote directory to save downloaded files")
	offlineCmd.AddCommand(offlineAddCmd)
	offlineCmd.AddCommand(offlineListCmd)
	offlineCmd.AddCommand(offlineRmCmd)
	rootCmd.AddCommand(offlineCmd)
}
```

- [ ] **Step 2: Build and verify**

```bash
go build -o 115driver ./cli/
```

- [ ] **Step 3: Commit**

```bash
git add cli/cmd/offline.go
git commit -m "feat(cli): add offline download commands (add/list/rm)"
```

---

## Task 13: Final Integration + Help Text Polish

**Files:**
- Modify: `cli/cmd/root.go` (add version, completion command)

- [ ] **Step 1: Add version flag to root command**

Add to `cli/cmd/root.go` `init()` function:

```go
rootCmd.Version = "0.1.0"
```

- [ ] **Step 2: Add completion subcommand**

Cobra has built-in completion generation. Add to `cli/cmd/root.go` `init()`:

```go
rootCmd.CompletionOptions.HiddenDefaultCmd = true
```

- [ ] **Step 3: Full build test**

```bash
go build -o 115driver ./cli/
./115driver --help
./115driver ls --help
./115driver upload --help
./115driver offline --help
```

Expected: all commands show proper help text.

- [ ] **Step 4: Run all tests to verify nothing broken**

```bash
go test ./...
```

Expected: all existing tests pass.

- [ ] **Step 5: Commit**

```bash
git add cli/
git commit -m "feat(cli): polish help text and add version flag"
```

---

## Dependency Summary

New dependencies to add via `go get`:
- `github.com/spf13/cobra` — CLI framework
- `github.com/spf13/viper` — Config management
- `github.com/olekukonez/tablewriter` — Table rendering
- `github.com/cheggaaa/pb/v3` — Progress bar

## Build Target

```bash
go build -o 115driver ./cli/
```

## Testing Strategy

Each task builds and compiles. Integration testing requires valid 115 credentials:
1. `115driver login` (QR code or cookie)
2. `115driver ls /`
3. `115driver mkdir /test-cli-dir`
4. `115driver upload ./test.txt /test-cli-dir/`
5. `115driver ls /test-cli-dir/`
6. `115driver search test`
7. `115driver rm /test-cli-dir/test.txt`
8. `115driver rm /test-cli-dir`

Each command tested in both human and `--json` mode.
