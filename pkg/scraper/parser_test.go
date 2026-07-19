package scraper

import (
	"testing"
	"time"
)

func TestIsTimeFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"7:30 PM", true},
		{"7:30 AM", true},
		{"19:30", true},
		{"2:45 pm", true},
		{"11:00 AM", true},
		{"Not a time", false},
		{"Hello world", false},
		{"", false},
	}

	for _, test := range tests {
		result := isTimeFormat(test.input)
		if result != test.expected {
			t.Errorf("isTimeFormat(%q) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestParseAvailability(t *testing.T) {
	html := `
	<html>
		<body>
			<button data-time="7:30 PM">7:30 PM</button>
			<button data-time="8:00 PM">8:00 PM</button>
			<button data-time="8:30 PM">8:30 PM</button>
		</body>
	</html>
	`

	slots, err := ParseAvailability(html, time.Now())
	if err != nil {
		t.Fatalf("ParseAvailability returned error: %v", err)
	}

	if len(slots) == 0 {
		t.Fatal("Expected to find time slots, but got none")
	}

	if len(slots) < 3 {
		t.Errorf("Expected at least 3 slots, got %d", len(slots))
	}
}
