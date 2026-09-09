package hidraw

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"
	"time"
)

func TestIocNumbersMatchKernelHeader(t *testing.T) {
	cases := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"HIDIOCSFEATURE(34)", hidiocSFeature(34), 0xC0224806},
		{"HIDIOCGFEATURE(34)", hidiocGFeature(34), 0xC0224807},
		{"HIDIOCSFEATURE(2)", hidiocSFeature(2), 0xC0024806},
		{"HIDIOCGINPUT(34)", hidiocGInput(34), 0xC022480A},
		{"HIDIOCSOUTPUT(34)", hidiocSOutput(34), 0xC022480B},
		{"HIDIOCGOUTPUT(34)", hidiocGOutput(34), 0xC022480C},
		{"HIDIOCSINPUT(34)", hidiocSInput(34), 0xC0224809},
		{"HIDIOCGRAWINFO", hidiocGRawInfo(), 0x80084803},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = 0x%08X, want 0x%08X", c.name, c.got, c.want)
		}
	}
}

func TestIocNumberVariesWithLength(t *testing.T) {
	a := hidiocSFeature(2)
	b := hidiocSFeature(34)
	if a == b {
		t.Fatalf("expected different ioctl numbers for different payload lengths, got 0x%08X for both", a)
	}
}

func TestParseUevent(t *testing.T) {
	content := "DRIVER=hid-generic\n" +
		"HID_ID=0003:0000187C:00000550\n" +
		"HID_NAME=Alienware AW-ELC\n" +
		"HID_PHYS=usb-0000:00:14.0-4/input0\n" +
		"HID_UNIQ=00.01\n" +
		"MODALIAS=hid:b0003g0001v0000187Cp00000550\n"

	info, ok := parseUevent(content)
	if !ok {
		t.Fatalf("expected HID_ID to be found")
	}
	if info.Vendor != 0x187c {
		t.Errorf("vendor = 0x%04x, want 0x187c", info.Vendor)
	}
	if info.Product != 0x0550 {
		t.Errorf("product = 0x%04x, want 0x0550", info.Product)
	}
	if info.Bus != "0003" {
		t.Errorf("bus = %q, want 0003", info.Bus)
	}
	if info.Name != "Alienware AW-ELC" {
		t.Errorf("name = %q, want Alienware AW-ELC", info.Name)
	}
}

func TestParseUeventMissingHidID(t *testing.T) {
	_, ok := parseUevent("DRIVER=hid-generic\nHID_NAME=Something\n")
	if ok {
		t.Fatalf("expected no match without HID_ID")
	}
}

func TestParseUeventMalformedHidID(t *testing.T) {
	_, ok := parseUevent("HID_ID=notanid\n")
	if ok {
		t.Fatalf("expected no match for malformed HID_ID")
	}
}

func TestOpenMissingDeviceReturnsNotFound(t *testing.T) {
	_, err := Open("/dev/does-not-exist-alienwarectl-test")
	if err == nil {
		t.Fatalf("expected an error opening a missing device")
	}
	var herr *Error
	if !asError(err, &herr) {
		t.Fatalf("error %v is not a *hidraw.Error", err)
	}
	if herr.Code() != CodeNotFound {
		t.Errorf("code = %q, want %q", herr.Code(), CodeNotFound)
	}
}

func asError(err error, target **Error) bool {
	e, ok := err.(*Error)
	if !ok {
		return false
	}
	*target = e
	return true
}

func TestFindOneByVIDPIDReturnsNotFoundWhenAbsent(t *testing.T) {
	_, err := FindOneByVIDPID(0xffff, 0xffff)
	if err == nil {
		return
	}
	herr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error %v is not a *hidraw.Error", err)
	}
	if herr.Code() != CodeNotFound && herr.Code() != CodeIO {
		t.Errorf("code = %q, want %q or %q", herr.Code(), CodeNotFound, CodeIO)
	}
}

func TestNodeOrdering(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "numeric suffix beats lexicographic",
			in:   []string{"/dev/hidraw10", "/dev/hidraw2"},
			want: []string{"/dev/hidraw2", "/dev/hidraw10"},
		},
		{
			name: "double digit run",
			in:   []string{"/dev/hidraw11", "/dev/hidraw3", "/dev/hidraw0", "/dev/hidraw10", "/dev/hidraw2"},
			want: []string{"/dev/hidraw0", "/dev/hidraw2", "/dev/hidraw3", "/dev/hidraw10", "/dev/hidraw11"},
		},
		{
			name: "unparseable name sorts after numbered nodes",
			in:   []string{"/dev/hidrawX", "/dev/hidraw10", "/dev/hidraw2"},
			want: []string{"/dev/hidraw2", "/dev/hidraw10", "/dev/hidrawX"},
		},
		{
			name: "two unparseable names keep string order",
			in:   []string{"/dev/hidrawB", "/dev/hidrawA"},
			want: []string{"/dev/hidrawA", "/dev/hidrawB"},
		},
		{
			name: "foreign prefix sorts after numbered nodes",
			in:   []string{"/dev/other0", "/dev/hidraw9"},
			want: []string{"/dev/hidraw9", "/dev/other0"},
		},
		{
			name: "empty suffix does not parse",
			in:   []string{"/dev/hidraw", "/dev/hidraw1"},
			want: []string{"/dev/hidraw1", "/dev/hidraw"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := append([]string(nil), c.in...)
			sort.Slice(got, func(i, j int) bool { return nodeLess(got[i], got[j]) })
			if !slices.Equal(got, c.want) {
				t.Fatalf("order = %v, want %v", got, c.want)
			}
		})
	}
}

func TestNodeIndex(t *testing.T) {
	cases := []struct {
		path string
		want uint64
		ok   bool
	}{
		{"/dev/hidraw0", 0, true},
		{"/dev/hidraw10", 10, true},
		{"/dev/hidrawX", 0, false},
		{"/dev/hidraw", 0, false},
		{"/dev/hidraw+2", 0, false},
		{"/dev/other2", 0, false},
	}
	for _, c := range cases {
		got, ok := nodeIndex(c.path)
		if ok != c.ok || got != c.want {
			t.Errorf("nodeIndex(%q) = %d, %v, want %d, %v", c.path, got, ok, c.want, c.ok)
		}
	}
}

func TestLockFileRejectsSecondHolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node")
	first := openForLock(t, path)
	defer first.Close()
	if err := lockFileWithin(first, 50*time.Millisecond, 5*time.Millisecond); err != nil {
		t.Fatalf("first lock failed: %v", err)
	}

	second := openForLock(t, path)
	defer second.Close()
	err := lockFileWithin(second, 50*time.Millisecond, 5*time.Millisecond)
	if err == nil {
		t.Fatalf("expected the second lock attempt to fail")
	}
	herr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error %v is not a *hidraw.Error", err)
	}
	if herr.Code() != CodeBusy {
		t.Errorf("code = %q, want %q", herr.Code(), CodeBusy)
	}
}

func TestLockFileSucceedsAfterRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node")
	first := openForLock(t, path)
	if err := lockFileWithin(first, 50*time.Millisecond, 5*time.Millisecond); err != nil {
		t.Fatalf("first lock failed: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	second := openForLock(t, path)
	defer second.Close()
	if err := lockFileWithin(second, 50*time.Millisecond, 5*time.Millisecond); err != nil {
		t.Fatalf("expected the lock to be free after close, got %v", err)
	}
}

func openForLock(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	return f
}

func withShortLockTimeout(t *testing.T) {
	t.Helper()
	prevTimeout, prevInterval := lockTimeout, lockInterval
	lockTimeout, lockInterval = 50*time.Millisecond, 5*time.Millisecond
	t.Cleanup(func() { lockTimeout, lockInterval = prevTimeout, prevInterval })
}

func TestOpenHoldsTheLockAgainstASecondOpen(t *testing.T) {
	withShortLockTimeout(t)
	path := filepath.Join(t.TempDir(), "node")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}

	first, err := Open(path)
	if err != nil {
		t.Fatalf("first open failed: %v", err)
	}

	if _, err := Open(path); err == nil {
		t.Fatalf("expected the second open to fail while the first holds the lock")
	} else {
		herr, ok := err.(*Error)
		if !ok {
			t.Fatalf("error %v is not a *hidraw.Error", err)
		}
		if herr.Code() != CodeBusy {
			t.Errorf("code = %q, want %q", herr.Code(), CodeBusy)
		}
	}

	if err := first.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	third, err := Open(path)
	if err != nil {
		t.Fatalf("expected the lock to be free after close, got %v", err)
	}
	third.Close()
}
