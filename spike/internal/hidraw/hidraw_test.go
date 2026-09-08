package hidraw

import "testing"

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
