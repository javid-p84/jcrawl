package scraper

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func mustPermitDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("bad date %q: %v", s, err)
	}
	return d
}

func TestMonthsBetween(t *testing.T) {
	monthKeys := func(months []time.Time) []string {
		keys := make([]string, len(months))
		for i, m := range months {
			keys[i] = m.Format("2006-01")
		}
		return keys
	}

	tests := []struct {
		name       string
		from, to   string
		wantMonths []string
	}{
		{"day-1 aligned range (jcrawl's own README example)", "2026-07-01", "2026-10-01", []string{"2026-07", "2026-08", "2026-09", "2026-10"}},
		{"single month", "2026-07-05", "2026-07-20", []string{"2026-07"}},
		// Regression test: advancing from an arbitrary day-of-month (e.g. the
		// 31st) can overflow past the target month when that month has fewer
		// days (Jan 31 + 1 month normalizes to Mar 3, skipping February
		// entirely). monthsBetween must not do this — every month the range
		// touches must be returned.
		{"31st crossing shorter months must not skip any month", "2026-01-31", "2026-03-01", []string{"2026-01", "2026-02", "2026-03"}},
		{"same regression, different months", "2026-03-31", "2026-05-01", []string{"2026-03", "2026-04", "2026-05"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := monthKeys(monthsBetween(mustPermitDate(t, tt.from), mustPermitDate(t, tt.to)))
			if !reflect.DeepEqual(got, tt.wantMonths) {
				t.Errorf("monthsBetween(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.wantMonths)
			}
		})
	}
}

func TestExtractPermitID(t *testing.T) {
	ps := NewPermitScraper()

	got, err := ps.ExtractPermitID("https://www.recreation.gov/permits/445860/registration/detailed-availability?date=2026-07-21&type=overnight-permit")
	if err != nil {
		t.Fatalf("ExtractPermitID: %v", err)
	}
	if got != "445860" {
		t.Errorf("ExtractPermitID = %q, want %q", got, "445860")
	}

	if _, err := ps.ExtractPermitID("https://www.recreation.gov/camping/campgrounds/232447/"); err == nil {
		t.Error("expected error extracting a permit ID from a campground URL")
	}
}

func TestIsPermitLink(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://www.recreation.gov/permits/445860/registration/detailed-availability?date=2026-07-21&type=overnight-permit", true},
		{"https://www.recreation.gov/camping/campgrounds/232447/", false},
		{"https://www.opentable.com/r/some-restaurant", false},
	}
	for _, tt := range tests {
		if got := isPermitLink(tt.url); got != tt.want {
			t.Errorf("isPermitLink(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

// Fixture captured from the real API response for permit 445860 (Mt. Whitney).
const permitAvailabilityFixture = `{
	"payload": {
		"2026-07-01": {
			"166": {"quota_usage_by_member_daily": {"total": 57, "remaining": 0}, "is_walkup": true},
			"406": {"quota_usage_by_member_daily": {"total": 87, "remaining": 5}, "is_walkup": true}
		},
		"2026-07-02": {
			"166": {"quota_usage_by_member_daily": {"total": 47, "remaining": 2}, "is_walkup": true}
		}
	}
}`

func TestParsePermitAvailability(t *testing.T) {
	var raw struct {
		Payload map[string]map[string]permitAvailabilityEntry `json:"payload"`
	}
	if err := json.Unmarshal([]byte(permitAvailabilityFixture), &raw); err != nil {
		t.Fatalf("failed to unmarshal fixture: %v", err)
	}

	result := parsePermitAvailability(raw.Payload)

	day1, ok := result["2026-07-01"]
	if !ok {
		t.Fatal("expected 2026-07-01 in result")
	}
	if day1["166"].Remaining != 0 || day1["166"].Total != 57 || !day1["166"].IsWalkup {
		t.Errorf("division 166 on 2026-07-01 = %+v, want total=57 remaining=0 is_walkup=true", day1["166"])
	}
	if day1["406"].Remaining != 5 {
		t.Errorf("division 406 remaining = %d, want 5", day1["406"].Remaining)
	}

	day2, ok := result["2026-07-02"]
	if !ok {
		t.Fatal("expected 2026-07-02 in result")
	}
	if day2["166"].Remaining != 2 {
		t.Errorf("division 166 on 2026-07-02 remaining = %d, want 2", day2["166"].Remaining)
	}
}

// Fixture captured from the real permitcontent API response for permit 445860.
const permitContentFixture = `{
	"payload": {
		"divisions": {
			"166": {"name": "Mt. Whitney Trail (Overnight)", "code": "JM35"},
			"406": {"name": "Mt. Whitney Day Use (All Routes)", "code": "JM34.5"}
		}
	}
}`

func TestParseDivisionNames(t *testing.T) {
	var raw struct {
		Payload struct {
			Divisions map[string]permitContentDivision `json:"divisions"`
		} `json:"payload"`
	}
	if err := json.Unmarshal([]byte(permitContentFixture), &raw); err != nil {
		t.Fatalf("failed to unmarshal fixture: %v", err)
	}

	names := parseDivisionNames(raw.Payload.Divisions)

	if names["166"] != "Mt. Whitney Trail (Overnight)" {
		t.Errorf("division 166 name = %q, want %q", names["166"], "Mt. Whitney Trail (Overnight)")
	}
	if names["406"] != "Mt. Whitney Day Use (All Routes)" {
		t.Errorf("division 406 name = %q, want %q", names["406"], "Mt. Whitney Day Use (All Routes)")
	}
}

func TestParseDivisionNames_FallsBackToCode(t *testing.T) {
	divisions := map[string]permitContentDivision{
		"999": {Name: "", Code: "ABC1"},
	}
	names := parseDivisionNames(divisions)
	if names["999"] != "ABC1" {
		t.Errorf("expected fallback to code %q, got %q", "ABC1", names["999"])
	}
}
