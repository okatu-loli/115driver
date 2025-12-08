package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SheltonZhu/115driver/pkg/driver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// FileTools holds file-related MCP tools
type FileTools struct {
	client *driver.Pan115Client
}

// NewFileTools creates a new FileTools instance
func NewFileTools(client *driver.Pan115Client) *FileTools {
	return &FileTools{
		client: client,
	}
}

// MkdirArgs defines arguments for mkdir tool
type MkdirArgs struct {
	ParentID string `json:"parent_id" jsonschema:"parent directory ID"`
	Name     string `json:"name" jsonschema:"name of the new directory"`
}

// DeleteArgs defines arguments for delete tool
type DeleteArgs struct {
	FileIDs []string `json:"file_ids" jsonschema:"IDs of files or directories to delete"`
}

// RenameArgs defines arguments for rename tool
type RenameArgs struct {
	FileID  string `json:"file_id" jsonschema:"ID of file or directory to rename"`
	NewName string `json:"new_name" jsonschema:"new name for the file or directory"`
}

// MoveArgs defines arguments for move tool
type MoveArgs struct {
	DirID   string   `json:"dir_id" jsonschema:"target directory ID"`
	FileIDs []string `json:"file_ids" jsonschema:"IDs of files or directories to move"`
}

// CopyArgs defines arguments for copy tool
type CopyArgs struct {
	DirID   string   `json:"dir_id" jsonschema:"target directory ID"`
	FileIDs []string `json:"file_ids" jsonschema:"IDs of files or directories to copy"`
}

// StatArgs defines arguments for stat tool
type StatArgs struct {
	FileID string `json:"file_id" jsonschema:"ID of file or directory to get info"`
}

// RegisterTools registers file-related tools with the MCP server
func (ft *FileTools) RegisterTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mkdir",
		Description: "Create a new directory",
	}, ft.mkdir)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete",
		Description: "Delete files or directories",
	}, ft.delete)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "rename",
		Description: "Rename a file or directory",
	}, ft.rename)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "move",
		Description: "Move files or directories to another directory",
	}, ft.move)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "copy",
		Description: "Copy files or directories to another directory",
	}, ft.copy)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "stat",
		Description: "Get detailed information about a file or directory",
	}, ft.stat)
}

func (ft *FileTools) mkdir(ctx context.Context, req *mcp.CallToolRequest, args MkdirArgs) (*mcp.CallToolResult, any, error) {
	dirID, err := ft.client.Mkdir(args.ParentID, args.Name)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Failed to create directory: %v", err),
				},
			},
			IsError: true,
		}, nil, nil
	}

	result := map[string]string{
		"directory_id": dirID,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Failed to serialize result: %v", err),
				},
			},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(resultJSON),
			},
		},
	}, nil, nil
}

func (ft *FileTools) delete(ctx context.Context, req *mcp.CallToolRequest, args DeleteArgs) (*mcp.CallToolResult, any, error) {
	if len(args.FileIDs) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "No file IDs provided",
				},
			},
			IsError: true,
		}, nil, nil
	}

	err := ft.client.Delete(args.FileIDs...)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Failed to delete files: %v", err),
				},
			},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: "Files deleted successfully",
			},
		},
	}, nil, nil
}

func (ft *FileTools) rename(ctx context.Context, req *mcp.CallToolRequest, args RenameArgs) (*mcp.CallToolResult, any, error) {
	err := ft.client.Rename(args.FileID, args.NewName)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Failed to rename file: %v", err),
				},
			},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: "File renamed successfully",
			},
		},
	}, nil, nil
}

func (ft *FileTools) move(ctx context.Context, req *mcp.CallToolRequest, args MoveArgs) (*mcp.CallToolResult, any, error) {
	if len(args.FileIDs) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "No file IDs provided",
				},
			},
			IsError: true,
		}, nil, nil
	}

	err := ft.client.Move(args.DirID, args.FileIDs...)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Failed to move files: %v", err),
				},
			},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: "Files moved successfully",
			},
		},
	}, nil, nil
}

func (ft *FileTools) copy(ctx context.Context, req *mcp.CallToolRequest, args CopyArgs) (*mcp.CallToolResult, any, error) {
	if len(args.FileIDs) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "No file IDs provided",
				},
			},
			IsError: true,
		}, nil, nil
	}

	err := ft.client.Copy(args.DirID, args.FileIDs...)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Failed to copy files: %v", err),
				},
			},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: "Files copied successfully",
			},
		},
	}, nil, nil
}

func (ft *FileTools) stat(ctx context.Context, req *mcp.CallToolRequest, args StatArgs) (*mcp.CallToolResult, any, error) {
	info, err := ft.client.Stat(args.FileID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Failed to get file info: %v", err),
				},
			},
			IsError: true,
		}, nil, nil
	}

	result := map[string]interface{}{
		"name":         info.Name,
		"pick_code":    info.PickCode,
		"sha1":         info.Sha1,
		"is_directory": info.IsDirectory,
		"file_count":   info.FileCount,
		"dir_count":    info.DirCount,
		"create_time":  info.CreateTime,
		"update_time":  info.UpdateTime,
		"parents":      info.Parents,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Failed to serialize result: %v", err),
				},
			},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(resultJSON),
			},
		},
	}, nil, nil
}