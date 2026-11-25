package stdlib

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
)

func setCSVTestACL(t *testing.T, path, attribute string) {
	t.Helper()
	const undefinedID = ^uint32(0)
	entries := []struct {
		tag, permissions uint16
		id               uint32
	}{
		{0x01, 6, undefinedID},         // owner: rw-
		{0x02, 4, uint32(os.Getuid())}, // a named user entry keeps the ACL extended
		{0x04, 0, undefinedID},         // owning group: ---
		{0x10, 4, undefinedID},         // ACL mask: r--
		{0x20, 0, undefinedID},         // other: ---
	}
	data := binary.LittleEndian.AppendUint32(nil, 2) // Linux POSIX ACL xattr version.
	for _, entry := range entries {
		data = binary.LittleEndian.AppendUint16(data, entry.tag)
		data = binary.LittleEndian.AppendUint16(data, entry.permissions)
		data = binary.LittleEndian.AppendUint32(data, entry.id)
	}
	if err := syscall.Setxattr(path, attribute, data, 0); err != nil {
		if errors.Is(err, syscall.ENOTSUP) {
			t.Skipf("filesystem does not support POSIX ACLs: %v", err)
		}
		t.Fatal(err)
	}
}

func TestCSVSinkPreservesAccessACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.csv")
	if err := os.WriteFile(path, []byte("old\n"), 0600); err != nil {
		t.Fatal(err)
	}
	setCSVTestACL(t, path, csvAccessACL)
	before, err := readCSVAccessACL(path)
	if err != nil || len(before) == 0 {
		t.Fatalf("test fixture ACL = %x, %v", before, err)
	}
	beforeInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	rows := runtime.List{runtime.Record{"total": runtime.Number(1)}}
	if _, err := sink([]runtime.Value{rows, runtime.String(path)}); err != nil {
		t.Fatal(err)
	}
	after, err := readCSVAccessACL(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("ACL changed from %x to %x: %v", before, after, err)
	}
	afterInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	oldStat, newStat := beforeInfo.Sys().(*syscall.Stat_t), afterInfo.Sys().(*syscall.Stat_t)
	if beforeInfo.Mode().Perm() != afterInfo.Mode().Perm() || oldStat.Uid != newStat.Uid || oldStat.Gid != newStat.Gid {
		t.Fatal("destination access mode or ownership changed")
	}
}

func TestCSVSinkDoesNotAddInheritedAccessACL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.csv")
	if err := os.WriteFile(path, []byte("old\n"), 0640); err != nil {
		t.Fatal(err)
	}
	setCSVTestACL(t, dir, "system.posix_acl_default")
	probe, err := os.CreateTemp(dir, "probe-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}
	if acl, err := readCSVAccessACL(probe.Name()); err != nil || len(acl) == 0 {
		t.Fatalf("test directory did not provide an inherited ACL: %x, %v", acl, err)
	}

	rows := runtime.List{runtime.Record{"total": runtime.Number(1)}}
	if _, err := sink([]runtime.Value{rows, runtime.String(path)}); err != nil {
		t.Fatal(err)
	}
	if acl, err := readCSVAccessACL(path); err != nil || acl != nil {
		t.Fatalf("sink added an access ACL: %x, %v", acl, err)
	}
}
