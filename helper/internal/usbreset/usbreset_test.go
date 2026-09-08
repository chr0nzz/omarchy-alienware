package usbreset

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestFindReturnsNotFoundOnEmptyBus(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	l := &Locator{SysRoot: root, DevRoot: filepath.Join(t.TempDir(), "bus")}
	_, err := l.find()
	if err == nil {
		t.Fatal("want an error when no device matches")
	}
	var oerr *Error
	if !errors.As(err, &oerr) || oerr.Code() != CodeNotFound {
		t.Fatalf("got %v, want a %q error", err, CodeNotFound)
	}
}

func TestFindSkipsNonMatchingDevices(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "1-1", "idVendor"), "8087\n")
	writeFile(t, filepath.Join(root, "1-1", "idProduct"), "0aaa\n")
	writeFile(t, filepath.Join(root, "1-1", "busnum"), "1\n")
	writeFile(t, filepath.Join(root, "1-1", "devnum"), "3\n")
	l := &Locator{SysRoot: root, DevRoot: filepath.Join(t.TempDir(), "bus")}
	_, err := l.find()
	if err == nil {
		t.Fatal("want an error when only unrelated devices are present")
	}
	var oerr *Error
	if !errors.As(err, &oerr) || oerr.Code() != CodeNotFound {
		t.Fatalf("got %v, want a %q error", err, CodeNotFound)
	}
}

func TestFindBuildsDevNodeFromBusAndDevNum(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "1-1", "idVendor"), "8087\n")
	writeFile(t, filepath.Join(root, "1-1", "idProduct"), "0aaa\n")
	writeFile(t, filepath.Join(root, "1-1", "busnum"), "1\n")
	writeFile(t, filepath.Join(root, "1-1", "devnum"), "2\n")
	writeFile(t, filepath.Join(root, "3-2", "idVendor"), VendorID+"\n")
	writeFile(t, filepath.Join(root, "3-2", "idProduct"), ProductID+"\n")
	writeFile(t, filepath.Join(root, "3-2", "busnum"), "3\n")
	writeFile(t, filepath.Join(root, "3-2", "devnum"), "17\n")
	devRoot := filepath.Join(t.TempDir(), "bus")
	l := &Locator{SysRoot: root, DevRoot: devRoot}
	node, err := l.find()
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	want := filepath.Join(devRoot, "003", "017")
	if node != want {
		t.Fatalf("node: got %q, want %q", node, want)
	}
}

func TestResetFailsCleanlyWhenNodeCannotBeOpened(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "3-2", "idVendor"), VendorID+"\n")
	writeFile(t, filepath.Join(root, "3-2", "idProduct"), ProductID+"\n")
	writeFile(t, filepath.Join(root, "3-2", "busnum"), "3\n")
	writeFile(t, filepath.Join(root, "3-2", "devnum"), "17\n")
	l := &Locator{SysRoot: root, DevRoot: filepath.Join(t.TempDir(), "no-such-bus")}
	_, err := l.Reset()
	if err == nil {
		t.Fatal("want an error when the usbfs node does not exist")
	}
	var oerr *Error
	if !errors.As(err, &oerr) || oerr.Code() != CodeFailed {
		t.Fatalf("got %v, want a %q error", err, CodeFailed)
	}
}
