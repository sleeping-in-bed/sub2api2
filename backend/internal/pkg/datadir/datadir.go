package datadir

import (
	"os"
	"strings"
)

const defaultContainerDir = "/app/data"

// Resolve returns the data directory for persistent application files.
// Priority: DATA_DIR env > /app/data (if writable) > current directory.
func Resolve() string {
	return resolve(os.Getenv("DATA_DIR"), defaultContainerDir)
}

func resolve(explicitDir, containerDir string) string {
	if dir := strings.TrimSpace(explicitDir); dir != "" {
		return dir
	}
	if dir := strings.TrimSpace(containerDir); dir != "" && isWritableDirectory(dir) {
		return dir
	}
	return "."
}

func isWritableDirectory(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	file, err := os.CreateTemp(dir, ".write-test-*")
	if err != nil {
		return false
	}
	name := file.Name()
	closeErr := file.Close()
	removeErr := os.Remove(name)
	return closeErr == nil && removeErr == nil
}
