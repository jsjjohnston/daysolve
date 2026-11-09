package solver

import (
	"fmt"
	"time"
)

var testZone = time.FixedZone("TEST", 60*60)

// at returns the "15:04" clock time on the test day, in the test zone.
func at(clock string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", "2026-11-10 "+clock, testZone)
	if err != nil {
		panic(err)
	}
	return t
}

func span(start, end string) Interval {
	return Interval{Start: at(start), End: at(end)}
}

func event(start, end string) FixedEvent {
	return FixedEvent{Title: "meeting", Start: at(start), End: at(end)}
}

// workday is 09:00 to 17:00 with no breaks and no buffer.
func workday() Day {
	return Day{Start: at("09:00"), End: at("17:00")}
}

// clocks renders intervals as "09:00-10:00" for compact comparison.
func clocks(in []Interval) []string {
	out := []string{}
	for _, i := range in {
		out = append(out, fmt.Sprintf("%s-%s", i.Start.Format("15:04"), i.End.Format("15:04")))
	}
	return out
}
