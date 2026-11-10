package clock

import (
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	for in, want := range map[string]Time{"09:00": {9, 0}, "00:00": {0, 0}, "23:59": {23, 59}, "14:20": {14, 20}} {
		got, err := Parse(in)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %v, %v; want %v", in, got, err, want)
		}
		if got.String() != in {
			t.Errorf("String() = %q, want %q", got.String(), in)
		}
	}
	for _, in := range []string{"", "9", "24:00", "09:60", "09:00:00", "nine"} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q): want an error", in)
		}
	}
}

func TestTimeBefore(t *testing.T) {
	if !(Time{9, 0}).Before(Time{9, 1}) || !(Time{9, 59}).Before(Time{10, 0}) {
		t.Error("earlier time should be before later time")
	}
	if (Time{9, 0}).Before(Time{9, 0}) || (Time{10, 0}).Before(Time{9, 59}) {
		t.Error("a time is not before itself or before an earlier time")
	}
}

func TestTimeText(t *testing.T) {
	var got Time
	if err := got.UnmarshalText([]byte("14:20")); err != nil || got != (Time{14, 20}) {
		t.Fatalf("UnmarshalText = %v, %v", got, err)
	}
	if b, _ := got.MarshalText(); string(b) != "14:20" {
		t.Errorf("MarshalText = %q", b)
	}
	if err := got.UnmarshalText([]byte("soon")); err == nil {
		t.Error("want an error for a bad time")
	}
}

func TestParseDate(t *testing.T) {
	got, err := ParseDate("2026-11-10")
	want := Date{2026, time.November, 10}
	if err != nil || got != want {
		t.Fatalf("ParseDate = %v, %v; want %v", got, err, want)
	}
	if got.String() != "2026-11-10" {
		t.Errorf("String() = %q", got.String())
	}
	for _, in := range []string{"", "2026-13-01", "10/11/2026", "2026-11-10T09:00:00Z"} {
		if _, err := ParseDate(in); err == nil {
			t.Errorf("ParseDate(%q): want an error", in)
		}
	}
}

func TestOn(t *testing.T) {
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		date    Date
		time    Time
		want    string // RFC 3339, or empty when an error is expected
		wantErr string
	}{
		{"winter", Date{2026, time.November, 10}, Time{9, 0}, "2026-11-10T09:00:00Z", ""},
		{"summer", Date{2026, time.July, 1}, Time{9, 0}, "2026-07-01T09:00:00+01:00", ""},
		{"midnight", Date{2026, time.November, 10}, Time{0, 0}, "2026-11-10T00:00:00Z", ""},
		{"before clocks go forward", Date{2026, time.March, 29}, Time{0, 30}, "2026-03-29T00:30:00Z", ""},
		{"skipped when clocks go forward", Date{2026, time.March, 29}, Time{1, 30}, "", "does not exist"},
		{"after clocks go forward", Date{2026, time.March, 29}, Time{2, 0}, "2026-03-29T02:00:00+01:00", ""},
		{"before clocks go back", Date{2026, time.October, 25}, Time{0, 30}, "2026-10-25T00:30:00+01:00", ""},
		{"twice when clocks go back", Date{2026, time.October, 25}, Time{1, 30}, "", "happens twice"},
		{"after clocks go back", Date{2026, time.October, 25}, Time{2, 0}, "2026-10-25T02:00:00Z", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := On(tt.date, tt.time, london)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("want error containing %q, got %v, %v", tt.wantErr, got, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Format(time.RFC3339) != tt.want {
				t.Errorf("got %s, want %s", got.Format(time.RFC3339), tt.want)
			}
			if got.Location() != london {
				t.Errorf("result is in %v, want %v", got.Location(), london)
			}
		})
	}
}
