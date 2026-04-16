package resolver

import (
	"fmt"
	"path"
	"strings"

	"github.com/SheltonZhu/115driver/pkg/driver"
)

const RootID = "0"

// TODO: add LRU cache for path resolution

func ResolveDir(client *driver.Pan115Client, remotePath string) (string, error) {
	if remotePath == "" || remotePath == "/" {
		return RootID, nil
	}

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

func ResolvePath(client *driver.Pan115Client, remotePath string) (string, bool, error) {
	if remotePath == "" || remotePath == "/" {
		return RootID, true, nil
	}

	// Try as directory first
	dirID, err := ResolveDir(client, remotePath)
	if err == nil && dirID != "" {
		return dirID, true, nil
	}

	// Try as file
	fileID, err := ResolveFile(client, remotePath)
	if err != nil {
		return "", false, fmt.Errorf("path not found: %s", remotePath)
	}
	return fileID, false, nil
}
