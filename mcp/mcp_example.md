# 115driver MCP Usage Example

## Introduction

115driver MCP is a Model Context Protocol compliant service that allows access to 115 cloud storage features through the standard MCP protocol.

## Quick Start

### 1. Compile the MCP server

```bash
go build -o mcp-server mcp/main.go
```

### 2. Run the MCP server

Valid 115 cloud cookies are required for authentication:

```bash
./mcp-server --cookie="UID=your_uid;CID=your_cid;SEID=your_seid"
```

### 3. Call Tools

Once the server is running, tools can be invoked via the JSON-RPC protocol. For example, to list the contents of the root directory, send the following JSON:

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "tools/call",
  "params": {
    "name": "listDirectory",
    "arguments": {
      "dir_id": "0"
    }
  }
}
```

Where [dir_id](file:///Users/sheltonzhu/github/115driver/pkg/driver/dir.go#L30-L30) of "0" represents the root directory.

### 4. Response Format

The server will return a response in a format similar to the following:

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "[{\"Id\":\"12345\",\"Name\":\"Documents\",\"Size\":0,\"Type\":1,\"CreateTime\":1234567890},{\"Id\":\"67890\",\"Name\":\"image.jpg\",\"Size\":1024,\"Type\":0,\"CreateTime\":1234567891}]"
      }
    ]
  }
}
```

## Tool List

Currently supported tools:

### listDirectory

Lists files and directories in the specified directory.

Parameters:
- [dir_id](file:///Users/sheltonzhu/github/115driver/pkg/driver/dir.go#L30-L30) (string): The ID of the directory to list, defaults to root directory "0"
- offset (int64, optional): Offset for pagination, defaults to 0
- limit (int64, optional): Number of items to return, defaults to all items

### mkdir

Create a new directory.

Parameters:
- parent_id (string, required): Parent directory ID
- name (string, required): Name of the new directory

### delete

Delete files or directories.

Parameters:
- file_ids ([]string, required): IDs of files or directories to delete

### rename

Rename a file or directory.

Parameters:
- file_id (string, required): ID of file or directory to rename
- new_name (string, required): New name for the file or directory

### move

Move files or directories to another directory.

Parameters:
- dir_id (string, required): Target directory ID
- file_ids ([]string, required): IDs of files or directories to move

### copy

Copy files or directories to another directory.

Parameters:
- dir_id (string, required): Target directory ID
- file_ids ([]string, required): IDs of files or directories to copy

### stat

Get detailed information about a file or directory.

Parameters:
- file_id (string, required): ID of file or directory to get info

### listRecycleBin

List items in the recycle bin.

Parameters:
- offset (string, optional): Offset for pagination, defaults to "0"
- limit (string, optional): Number of items to return, defaults to "40"

### revertRecycleBin

Revert items from the recycle bin.

Parameters:
- item_ids ([]string, required): IDs of items to revert

### cleanRecycleBin

Clean items from the recycle bin.

Parameters:
- password (string, required): Password for cleaning recycle bin
- item_ids ([]string, required): IDs of items to clean

### getShareSnap

Gets shared files and directories snapshot information.

Parameters:
- share_code (string, required): Share code
- receive_code (string, required): Receive code
- dir_id (string, optional): Directory ID to list, defaults to root directory
- offset (int, optional): Offset for pagination, defaults to 0
- limit (int, optional): Number of items to return, defaults to 20

## Notes

1. Valid cookies must be provided for authentication
2. The file list in the response is returned as a text content in JSON string format
3. The Type field indicates the file type: typically 1 for directories and 0 for files
4. All tools follow the standard MCP tool calling conventions