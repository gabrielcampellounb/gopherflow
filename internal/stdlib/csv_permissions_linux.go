package stdlib

import (
	"errors"
	"os"
	"syscall"
)

const csvAccessACL = "system.posix_acl_access"

func preserveCSVPermissions(file *os.File, path string, existing os.FileInfo) error {
	acl, err := readCSVAccessACL(path)
	if err != nil {
		return err
	}
	tempInfo, err := file.Stat()
	if err != nil {
		return err
	}
	original := existing.Sys().(*syscall.Stat_t)
	temporary := tempInfo.Sys().(*syscall.Stat_t)
	if original.Uid != temporary.Uid || original.Gid != temporary.Gid {
		if err := file.Chown(int(original.Uid), int(original.Gid)); err != nil {
			return err
		}
	}
	if acl != nil {
		// Linux applies ACL entries and their corresponding mode bits together.
		return syscall.Setxattr(file.Name(), csvAccessACL, acl, 0)
	}
	// A directory's default ACL may have added entries to the temporary file.
	// Remove them before chmod can grant access that the original did not have.
	if err := syscall.Removexattr(file.Name(), csvAccessACL); err != nil && !aclAbsent(err) {
		return err
	}
	return file.Chmod(existing.Mode().Perm())
}

func readCSVAccessACL(path string) ([]byte, error) {
	size, err := syscall.Getxattr(path, csvAccessACL, nil)
	if aclAbsent(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	acl := make([]byte, size)
	length, err := syscall.Getxattr(path, csvAccessACL, acl)
	if err != nil {
		return nil, err
	}
	return acl[:length], nil
}

func aclAbsent(err error) bool {
	return errors.Is(err, syscall.ENODATA) || errors.Is(err, syscall.ENOTSUP)
}
