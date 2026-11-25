//go:build !linux

package stdlib

import "os"

func preserveCSVPermissions(file *os.File, _ string, existing os.FileInfo) error {
	return file.Chmod(existing.Mode().Perm())
}
