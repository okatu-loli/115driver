# 115driver Project Context

## Project Overview

The 115driver is a Go-based cloud storage driver package for the 115 cloud service (a Chinese cloud storage provider). It provides a comprehensive API for interacting with 115 cloud storage, including file operations, user authentication, and advanced features like offline downloads.

The project is written in Go (version 1.23.0+) and follows modern Go practices. It includes both traditional API functionality and Model Context Protocol (MCP) server capabilities for AI integration.

## Key Features

### Authentication & User Management
- Cookie-based authentication (UID, CID, SEID, KID)
- QR code login support
- User information retrieval
- Session management

### File Operations
- File listing, upload, download
- File operations: rename, move, copy, delete
- Directory creation and management
- Rapid upload with SHA1-based deduplication
- Search functionality
- File metadata management (labels, star ratings)

### Advanced Features
- Offline download management
- Recycle bin operations (list, restore, clean)
- Share code functionality
- Upload via OSS (Aliyun Object Storage Service)
- Multipart upload for large files

### MCP (Model Context Protocol) Integration
- Full MCP server implementation
- Multiple tool categories: directory, file, recycle, share, search, offline
- Standardized JSON-RPC interface for AI agents

## Architecture

### Core Components
- **Pan115Client**: Main client struct managing HTTP connections and authentication
- **Crypto Package**: Custom encryption implementations (ECDH, AES, custom protocols)
- **MCP Server**: Standardized protocol server for AI integration
- **API Layer**: HTTP client wrapper with 115-specific authentication

### Crypto Implementation
The project implements sophisticated encryption including:
- ECDH key exchange for secure communication
- Custom XOR-based obfuscation
- AES encryption with ECB and CBC modes
- LZ4 compression for encrypted payloads

### Dependencies
- `github.com/go-resty/resty/v2`: HTTP client with middleware support
- `github.com/modelcontextprotocol/go-sdk`: MCP protocol implementation
- `github.com/aliyun/aliyun-oss-go-sdk`: Aliyun OSS integration
- `github.com/aead/ecdh`: Elliptic curve Diffie-Hellman key exchange
- `github.com/andreburgaud/crypt2go`: Cryptographic utilities

## Building and Running

### Prerequisites
- Go 1.23.0 or higher
- Access to 115 cloud service (requires valid account credentials)

### Building the Project
```bash
# Clone the repository
git clone https://github.com/SheltonZhu/115driver.git
cd 115driver

# Install dependencies
go mod tidy

# Build the main package
go build ./cmd/...

# Build the MCP server
go build -o mcp-server ./mcp/main.go
```

### Running the MCP Server
```bash
# Run with cookie authentication
./mcp-server --cookie="UID=your_uid;CID=your_cid;SEID=your_seid;KID=your_kid"
```

### Using as a Library
```go
import "github.com/SheltonZhu/115driver/pkg/driver"

// Initialize client with credentials
cr := &driver.Credential{
    UID: "xxx",
    CID: "xxx", 
    SEID: "xxx",
    KID: "xxx",
}

client := driver.Default().ImportCredential(cr)
if err := client.LoginCheck(); err != nil {
    log.Fatalf("login error: %s", err)
}
```

## Development Conventions

### Code Structure
- `/pkg/driver`: Core driver functionality
- `/mcp`: Model Context Protocol server implementation
- `/pkg/crypto`: Encryption and cryptographic utilities
- `/mcp/server/tools`: MCP tool implementations

### Testing
- Unit tests using Go's standard testing package
- Integration tests for API functionality
- Example tests demonstrating usage patterns

### Error Handling
- Comprehensive error types defined in `error.go`
- Standardized error responses from 115 API
- Proper error wrapping with context

## Special Notes

### Security Considerations
- Credentials are stored as cookies and must be handled securely
- Communication uses custom encryption protocols
- API tokens have limited lifespans and require refresh

### Performance Optimizations
- Rapid upload using SHA1-based deduplication
- Multipart upload for large files
- Internal network optimizations for Aliyun OSS

### MCP Protocol Support
The project includes a full MCP server implementation, allowing AI agents to interact with 115 cloud storage through standardized JSON-RPC calls. This makes it suitable for integration with AI development environments and tools.