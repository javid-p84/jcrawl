package scraper

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
	"time"
)

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("bad date %q: %v", s, err)
	}
	return d
}

// This is the scenario from the feature request: day_preference = Fri, Sat,
// Sun should only ever produce Friday as a candidate start date, since
// Saturday and Sunday are continuations of the same run, not separate starts.
func TestIsStartOfPreferredRun_FridaySaturdaySunday(t *testing.T) {
	// 2024-07-05 is a Friday
	fri := mustDate(t, "2024-07-05")
	sat := mustDate(t, "2024-07-06")
	sun := mustDate(t, "2024-07-07")
	thu := mustDate(t, "2024-07-04")
	mon := mustDate(t, "2024-07-08")

	dayPreference := []int{5, 6, 0} // Fri, Sat, Sun

	tests := []struct {
		name string
		date time.Time
		want bool
	}{
		{"Friday is a start", fri, true},
		{"Saturday is not a start (continuation)", sat, false},
		{"Sunday is not a start (continuation)", sun, false},
		{"Thursday is not preferred at all", thu, false},
		{"Monday is not preferred at all", mon, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isStartOfPreferredRun(tt.date, dayPreference); got != tt.want {
				t.Errorf("isStartOfPreferredRun(%s) = %v, want %v", tt.date.Format("2006-01-02 Mon"), got, tt.want)
			}
		})
	}
}

// A run that wraps around the week boundary (Saturday, Sunday) should still
// correctly identify Saturday — not Sunday — as the start.
func TestIsStartOfPreferredRun_WeekWraparound(t *testing.T) {
	sat := mustDate(t, "2024-07-06")
	sun := mustDate(t, "2024-07-07")
	dayPreference := []int{6, 0} // Sat, Sun

	if !isStartOfPreferredRun(sat, dayPreference) {
		t.Error("expected Saturday to be the start of a Sat/Sun run")
	}
	if isStartOfPreferredRun(sun, dayPreference) {
		t.Error("expected Sunday NOT to be a separate start when Saturday is also preferred")
	}
}

func TestIsStartOfPreferredRun_EmptyPreferenceAllowsEveryDay(t *testing.T) {
	for _, d := range []time.Time{mustDate(t, "2024-07-01"), mustDate(t, "2024-07-02")} {
		if !isStartOfPreferredRun(d, nil) {
			t.Errorf("expected every day to qualify when day_preference is empty, got false for %s", d)
		}
	}
}

func TestFindConsecutiveAvailability(t *testing.T) {
	start := mustDate(t, "2024-07-05") // Friday

	sites := map[string]SiteAvailability{
		"A": {SiteID: "A", SiteName: "Site A", AvailableDates: map[string]bool{
			"2024-07-05": true, "2024-07-06": true, "2024-07-07": true, // full 3-night block
		}},
		"B": {SiteID: "B", SiteName: "Site B", AvailableDates: map[string]bool{
			"2024-07-05": true, "2024-07-06": true, // missing Sunday
		}},
		"C": {SiteID: "C", SiteName: "Site C", AvailableDates: map[string]bool{
			"2024-07-06": true, "2024-07-07": true, "2024-07-08": true, // wrong window
		}},
	}

	got := findConsecutiveAvailability(sites, start, 3)
	sort.Strings(got)
	want := []string{"Site A"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("findConsecutiveAvailability = %v, want %v", got, want)
	}

	// With nights=1, both A and B should match (they're available on the 5th)
	got1 := findConsecutiveAvailability(sites, start, 1)
	sort.Strings(got1)
	want1 := []string{"Site A", "Site B"}
	if !reflect.DeepEqual(got1, want1) {
		t.Errorf("findConsecutiveAvailability(nights=1) = %v, want %v", got1, want1)
	}
}

func TestParseMonthAvailability(t *testing.T) {
	raw := `{
		"campsites": {
			"site-1": {
				"site": "Loop A, Site 12",
				"availabilities": {
					"2024-07-05T00:00:00Z": 1,
					"2024-07-06T00:00:00Z": 1,
					"2024-07-07T00:00:00Z": 0
				}
			},
			"site-2": {
				"site": "Loop B, Site 3",
				"availabilities": {
					"2024-07-05T00:00:00Z": 0
				}
			}
		}
	}`

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("failed to unmarshal fixture: %v", err)
	}

	result := parseMonthAvailability(data)

	site1, ok := result["site-1"]
	if !ok {
		t.Fatal("expected site-1 in result")
	}
	if site1.SiteName != "Loop A, Site 12" {
		t.Errorf("site-1 name = %q, want %q", site1.SiteName, "Loop A, Site 12")
	}
	if !site1.AvailableDates["2024-07-05"] || !site1.AvailableDates["2024-07-06"] {
		t.Error("expected site-1 available on 2024-07-05 and 2024-07-06")
	}
	if site1.AvailableDates["2024-07-07"] {
		t.Error("expected site-1 NOT available on 2024-07-07 (status 0)")
	}

	site2, ok := result["site-2"]
	if !ok {
		t.Fatal("expected site-2 in result")
	}
	if len(site2.AvailableDates) != 0 {
		t.Errorf("expected site-2 to have no available dates, got %v", site2.AvailableDates)
	}
}
