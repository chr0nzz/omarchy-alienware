package elc

import "strings"

type RegionID string

const (
	RegionPower      RegionID = "power"
	RegionLogo       RegionID = "logo"
	RegionRingTop    RegionID = "ring-top"
	RegionRingBottom RegionID = "ring-bottom"
)

type Region struct {
	ID      RegionID
	Name    string
	ZoneIDs []uint8
}

var Regions = []Region{
	{ID: RegionPower, Name: "Power button", ZoneIDs: []uint8{0}},
	{ID: RegionLogo, Name: "Lid logo", ZoneIDs: []uint8{1}},
	{ID: RegionRingTop, Name: "Ring top", ZoneIDs: []uint8{8, 9, 10, 11, 12, 13, 14, 15}},
	{ID: RegionRingBottom, Name: "Ring bottom", ZoneIDs: []uint8{16, 17, 18, 19, 20, 21, 22, 23}},
}

func RegionIDs() []string {
	ids := make([]string, len(Regions))
	for i, r := range Regions {
		ids[i] = string(r.ID)
	}
	return ids
}

func AllZoneIDs() []uint8 {
	var ids []uint8
	for _, r := range Regions {
		ids = append(ids, r.ZoneIDs...)
	}
	return ids
}

func ResolveRegion(id string) (Region, error) {
	for _, r := range Regions {
		if string(r.ID) == id {
			return r, nil
		}
	}
	return Region{}, newError(CodeBadRequest, "region %q is not valid, want one of %s", id, strings.Join(RegionIDs(), ", "))
}
