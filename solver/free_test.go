package solver

import (
	"slices"
	"testing"
	"time"
)

func TestFreeIntervals(t *testing.T) {
	lunch := []Interval{span("12:00", "13:00")}
	tests := []struct {
		name   string
		day    Day
		events []FixedEvent
		now    time.Time
		want   []string
	}{
		{"empty day", workday(), nil, time.Time{}, []string{"09:00-17:00"}},
		{"meeting in the middle", workday(), []FixedEvent{event("10:00", "11:00")}, time.Time{}, []string{"09:00-10:00", "11:00-17:00"}},
		{"break", Day{Start: at("09:00"), End: at("17:00"), Breaks: lunch}, nil, time.Time{}, []string{"09:00-12:00", "13:00-17:00"}},
		{
			"overlapping meetings",
			workday(),
			[]FixedEvent{event("10:00", "11:30"), event("11:00", "12:00")},
			time.Time{},
			[]string{"09:00-10:00", "12:00-17:00"},
		},
		{"fully booked", workday(), []FixedEvent{event("08:00", "18:00")}, time.Time{}, []string{}},
		{"meeting outside working hours", workday(), []FixedEvent{event("07:00", "08:00")}, time.Time{}, []string{"09:00-17:00"}},
		{"meeting with no duration", workday(), []FixedEvent{event("10:00", "10:00")}, time.Time{}, []string{"09:00-17:00"}},
		{
			"buffer widens meetings",
			Day{Start: at("09:00"), End: at("17:00"), Buffer: 10 * time.Minute},
			[]FixedEvent{event("10:00", "11:00")},
			time.Time{},
			[]string{"09:00-09:50", "11:10-17:00"},
		},
		{
			"buffer does not widen breaks",
			Day{Start: at("09:00"), End: at("17:00"), Breaks: lunch, Buffer: 10 * time.Minute},
			nil,
			time.Time{},
			[]string{"09:00-12:00", "13:00-17:00"},
		},
		{"now before the day", workday(), nil, at("07:00"), []string{"09:00-17:00"}},
		{"now during the day", workday(), nil, at("14:20"), []string{"14:20-17:00"}},
		{"now during a meeting", workday(), []FixedEvent{event("14:00", "15:00")}, at("14:20"), []string{"15:00-17:00"}},
		{"now after the day", workday(), nil, at("18:00"), []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clocks(freeIntervals(tt.day, tt.events, tt.now)); !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
