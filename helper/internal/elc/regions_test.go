package elc

import (
	"reflect"
	"strings"
	"testing"
)

func TestResolveRegionKnownIDs(t *testing.T) {
	cases := map[RegionID][]uint8{
		RegionPower:      {0},
		RegionLogo:       {1},
		RegionRingTop:    {8, 9, 10, 11, 12, 13, 14, 15},
		RegionRingBottom: {16, 17, 18, 19, 20, 21, 22, 23},
	}
	for id, wantZones := range cases {
		region, err := ResolveRegion(string(id))
		if err != nil {
			t.Fatalf("ResolveRegion(%q) error = %v", id, err)
		}
		if region.ID != id {
			t.Errorf("ResolveRegion(%q).ID = %q, want %q", id, region.ID, id)
		}
		if !reflect.DeepEqual(region.ZoneIDs, wantZones) {
			t.Errorf("ResolveRegion(%q).ZoneIDs = %v, want %v", id, region.ZoneIDs, wantZones)
		}
	}
}

func TestResolveRegionUnknownIDNamesValidOnes(t *testing.T) {
	_, err := ResolveRegion("keyboard")
	if err == nil {
		t.Fatalf("expected an error for an unknown region id")
	}
	eerr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error %v is not an *elc.Error", err)
	}
	if eerr.Code() != CodeBadRequest {
		t.Errorf("code = %q, want %q", eerr.Code(), CodeBadRequest)
	}
	for _, id := range RegionIDs() {
		if !strings.Contains(eerr.Error(), id) {
			t.Errorf("error message %q does not name valid region %q", eerr.Error(), id)
		}
	}
}

func TestAllZoneIDsCoversEveryRegionInOrder(t *testing.T) {
	want := []uint8{0, 1, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}
	got := AllZoneIDs()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AllZoneIDs() = %v, want %v", got, want)
	}
}

func TestRegionDisplayNames(t *testing.T) {
	want := map[RegionID]string{
		RegionPower:      "Power button",
		RegionLogo:       "Lid logo",
		RegionRingTop:    "Ring top",
		RegionRingBottom: "Ring bottom",
	}
	for _, r := range Regions {
		if r.Name != want[r.ID] {
			t.Errorf("region %q name = %q, want %q", r.ID, r.Name, want[r.ID])
		}
	}
}
