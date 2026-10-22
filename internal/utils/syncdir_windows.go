//go:build windows

package utils

// SyncDirectory is a no-op on Windows, which does not support directory fsync.
// The checkpoint file itself is synced before its atomic rename.
func SyncDirectory(path string) error { return nil }
