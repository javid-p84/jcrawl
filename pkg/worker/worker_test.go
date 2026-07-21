package worker

import (
	"testing"
	"time"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
)

func TestBestMatch(t *testing.T) {
	if got := bestMatch(nil); got != nil {
		t.Errorf("bestMatch(nil) = %v, want nil", got)
	}

	d := func(s string) time.Time {
		t.Helper()
		parsed, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatalf("bad date %q: %v", s, err)
		}
		return parsed
	}

	availabilities := []models.Availability{
		{Date: d("2024-07-12"), Time: "Site C"},
		{Date: d("2024-07-05"), Time: "Site A"}, // soonest — should win
		{Date: d("2024-07-06"), Time: "Site B"},
	}

	got := bestMatch(availabilities)
	if got == nil {
		t.Fatal("expected a match, got nil")
	}
	if got.Time != "Site A" {
		t.Errorf("bestMatch picked %q, want %q (soonest date)", got.Time, "Site A")
	}
}
