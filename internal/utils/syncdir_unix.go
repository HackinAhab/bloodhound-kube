//go:build !windows

package utils

import "os"

// SyncDirectory persists directory entries after an atomic checkpoint rename.
func SyncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
