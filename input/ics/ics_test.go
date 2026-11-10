package ics

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

func london(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

// calendar wraps event bodies in a VCALENDAR, with CRLF line endings.
func calendar(events ...string) string {
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//test//EN\r\n")
	for _, e := range events {
		b.WriteString("BEGIN:VEVENT\r\n")
		for _, line := range strings.Split(strings.TrimSpace(e), "\n") {
			b.WriteString(strings.TrimRight(line, " \t") + "\r\n")
		}
		b.WriteString("END:VEVENT\r\n")
	}
	b.WriteString("END:VCALENDAR\r\n")
	return b.String()
}

// read renders events as "Title 2026-11-10T09:00:00Z 2026-11-10T10:00:00Z",
// with " recurring" added when set.
func read(t *testing.T, in string) []string {
	t.Helper()
	events, err := Read(strings.NewReader(in), london(t))
	if err != nil {
		t.Fatal(err)
	}
	out := []string{}
	for _, e := range events {
		s := fmt.Sprintf("%s %s %s", e.Title, e.Start.Format(time.RFC3339), e.End.Format(time.RFC3339))
		if e.Recurring {
			s += " recurring"
		}
		out = append(out, s)
	}
	return out
}

func TestRead(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"no events", calendar(), []string{}},
		{
			"UTC times",
			calendar("SUMMARY:Stand-up\nDTSTART:20261110T100000Z\nDTEND:20261110T101500Z"),
			[]string{"Stand-up 2026-11-10T10:00:00Z 2026-11-10T10:15:00Z"},
		},
		{
			"TZID times are converted to the given zone",
			calendar("SUMMARY:Paris call\nDTSTART;TZID=Europe/Paris:20261110T110000\nDTEND;TZID=Europe/Paris:20261110T120000"),
			[]string{"Paris call 2026-11-10T10:00:00Z 2026-11-10T11:00:00Z"},
		},
		{
			"quoted TZID",
			calendar("SUMMARY:Call\nDTSTART;TZID=\"Europe/Paris\":20261110T110000\nDTEND;TZID=\"Europe/Paris\":20261110T120000"),
			[]string{"Call 2026-11-10T10:00:00Z 2026-11-10T11:00:00Z"},
		},
		{
			"floating times are read in the given zone",
			calendar("SUMMARY:Gym\nDTSTART:20260701T180000\nDTEND:20260701T190000"),
			[]string{"Gym 2026-07-01T18:00:00+01:00 2026-07-01T19:00:00+01:00"},
		},
		{
			"DURATION instead of DTEND",
			calendar("SUMMARY:Review\nDTSTART:20261110T140000Z\nDURATION:PT1H30M"),
			[]string{"Review 2026-11-10T14:00:00Z 2026-11-10T15:30:00Z"},
		},
		{
			"neither DTEND nor DURATION gives an event with no length",
			calendar("SUMMARY:Reminder\nDTSTART:20261110T140000Z"),
			[]string{"Reminder 2026-11-10T14:00:00Z 2026-11-10T14:00:00Z"},
		},
		{
			"sorted by start",
			calendar(
				"SUMMARY:Later\nDTSTART:20261110T150000Z\nDTEND:20261110T160000Z",
				"SUMMARY:Earlier\nDTSTART:20261110T090000Z\nDTEND:20261110T100000Z",
			),
			[]string{
				"Earlier 2026-11-10T09:00:00Z 2026-11-10T10:00:00Z",
				"Later 2026-11-10T15:00:00Z 2026-11-10T16:00:00Z",
			},
		},
		{
			"all-day events are left out",
			calendar(
				"SUMMARY:Holiday\nDTSTART;VALUE=DATE:20261110\nDTEND;VALUE=DATE:20261111",
				"SUMMARY:Birthday\nDTSTART:20261110",
			),
			[]string{},
		},
		{
			"cancelled and transparent events are left out",
			calendar(
				"SUMMARY:Called off\nSTATUS:CANCELLED\nDTSTART:20261110T100000Z\nDTEND:20261110T110000Z",
				"SUMMARY:Focus time\nTRANSP:TRANSPARENT\nDTSTART:20261110T100000Z\nDTEND:20261110T110000Z",
				"SUMMARY:Real\nSTATUS:CONFIRMED\nTRANSP:OPAQUE\nDTSTART:20261110T100000Z\nDTEND:20261110T110000Z",
			),
			[]string{"Real 2026-11-10T10:00:00Z 2026-11-10T11:00:00Z"},
		},
		{
			"recurring events are flagged",
			calendar(
				"SUMMARY:Weekly sync\nDTSTART:20261103T100000Z\nDTEND:20261103T103000Z\nRRULE:FREQ=WEEKLY;BYDAY=TU",
				"SUMMARY:Odd dates\nDTSTART:20261104T100000Z\nDTEND:20261104T103000Z\nRDATE:20261111T100000Z",
			),
			[]string{
				"Weekly sync 2026-11-03T10:00:00Z 2026-11-03T10:30:00Z recurring",
				"Odd dates 2026-11-04T10:00:00Z 2026-11-04T10:30:00Z recurring",
			},
		},
		{
			"folded lines, escapes and a missing title",
			calendar(
				"SUMMARY:Budget\\, plans\\; and a very long\n  title\\nsecond line\nDTSTART:20261110T100000Z\nDTEND:20261110T110000Z",
				"DTSTART:20261110T120000Z\nDTEND:20261110T130000Z",
			),
			[]string{
				"Budget, plans; and a very long title second line 2026-11-10T10:00:00Z 2026-11-10T11:00:00Z",
				"(untitled) 2026-11-10T12:00:00Z 2026-11-10T13:00:00Z",
			},
		},
		{
			"an alarm inside an event does not change it",
			calendar("SUMMARY:Dentist\nDTSTART:20261110T100000Z\nDTEND:20261110T110000Z\nBEGIN:VALARM\nTRIGGER:-PT15M\nSUMMARY:Alarm\nDURATION:PT5M\nEND:VALARM"),
			[]string{"Dentist 2026-11-10T10:00:00Z 2026-11-10T11:00:00Z"},
		},
		{
			"bare newlines, lower-case names and stray lines",
			"BEGIN:VCALENDAR\nthis line has no colon\nbegin:vevent\nsummary:Loose\ndtstart:20261110T100000Z\ndtend:20261110T110000Z\nend:vevent\nEND:VCALENDAR\n",
			[]string{"Loose 2026-11-10T10:00:00Z 2026-11-10T11:00:00Z"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := read(t, tt.in); !slices.Equal(got, tt.want) {
				t.Errorf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestReadReturnsTimesInTheGivenZone(t *testing.T) {
	loc := london(t)
	events, err := Read(strings.NewReader(calendar("SUMMARY:A\nDTSTART:20261110T100000Z\nDTEND:20261110T110000Z")), loc)
	if err != nil {
		t.Fatal(err)
	}
	if events[0].Start.Location() != loc || events[0].End.Location() != loc {
		t.Errorf("times are in %v and %v, want %v", events[0].Start.Location(), events[0].End.Location(), loc)
	}
}

func TestReadErrors(t *testing.T) {
	tests := []struct {
		name, in, wantErr string
	}{
		{"not a calendar", "hello\nworld\n", "not an iCalendar file"},
		{"unfinished event", "BEGIN:VCALENDAR\nBEGIN:VEVENT\nSUMMARY:Cut off\n", `event "Cut off": no END:VEVENT`},
		{"no start", calendar("SUMMARY:Lost"), `event "Lost": no DTSTART`},
		{"bad start", calendar("SUMMARY:A\nDTSTART:20261110T25"), `DTSTART: "20261110T25" is not a date-time`},
		{"bad UTC start", calendar("SUMMARY:A\nDTSTART:2026Z"), `DTSTART: "2026Z" is not a date-time`},
		{"bad end", calendar("SUMMARY:A\nDTSTART:20261110T100000Z\nDTEND:soonZ"), "DTEND:"},
		{"unknown zone", calendar("SUMMARY:A\nDTSTART;TZID=GMT Standard Time:20261110T100000"), `time zone "GMT Standard Time" is not an IANA zone name`},
		{"bad duration", calendar("SUMMARY:A\nDTSTART:20261110T100000Z\nDURATION:1h"), `DURATION: "1h" is not a duration`},
		{"negative duration", calendar("SUMMARY:A\nDTSTART:20261110T100000Z\nDURATION:-PT1H"), "DURATION:"},
		{"ends before it starts", calendar("SUMMARY:A\nDTSTART:20261110T100000Z\nDTEND:20261110T090000Z"), "ends before it starts"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Read(strings.NewReader(tt.in), london(t))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	for in, want := range map[string]time.Duration{
		"PT1H":       time.Hour,
		"PT30M":      30 * time.Minute,
		"PT1H30M":    90 * time.Minute,
		"PT45S":      45 * time.Second,
		"P1D":        24 * time.Hour,
		"P1W":        7 * 24 * time.Hour,
		"P1DT2H":     26 * time.Hour,
		"+PT15M":     15 * time.Minute,
		"pt1h":       time.Hour,
		"P1W2DT3H4M": (9*24+3)*time.Hour + 4*time.Minute,
	} {
		got, err := parseDuration(in)
		if err != nil || got != want {
			t.Errorf("parseDuration(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "1H", "-PT1H", "PT1X", "P99999999999999999999D"} {
		if got, err := parseDuration(in); err == nil {
			t.Errorf("parseDuration(%q) = %v; want an error", in, got)
		}
	}
}
