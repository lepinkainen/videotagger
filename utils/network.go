package utils

import (
	"path/filepath"
	"strings"
)

// IsNetworkDrive detects if a file path is on a network-mounted drive
func IsNetworkDrive(filePath string) bool {
	// Check Windows UNC paths first, before converting to absolute path
	if strings.HasPrefix(filePath, "//") || strings.HasPrefix(filePath, "\\\\") {
		return true
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}

	// Check common network mount prefixes on different platforms
	networkPrefixes := []string{
		"/mnt/",     // Linux NFS/SMB mounts
		"/media/",   // Linux removable/network media
		"/Volumes/", // macOS network volumes
	}

	for _, prefix := range networkPrefixes {
		if strings.HasPrefix(absPath, prefix) {
			return true
		}
	}

	// Check for network filesystem indicators in the path
	lowerPath := strings.ToLower(absPath)
	networkIndicators := []string{
		"nfs", "cifs", "smb", "webdav", "ftp", "sftp",
	}

	for _, indicator := range networkIndicators {
		if strings.Contains(lowerPath, indicator) {
			return true
		}
	}

	return false
}
