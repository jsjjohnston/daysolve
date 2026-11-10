// Package clock handles wall-clock times and calendar dates that have no
// location yet, and turns them into instants in a location.
package clock

import (
	"fmt"
	"time"
)

// Time is a time of day at minute precision.
type Time struct {
	Hour   int
	Minute int
}

// Parse reads a 24-hour time such as "09:00".
func Parse(s string) (Time, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return Time{}, fmt.Errorf("%q is not a time such as \"09:00\"", s)
	}
	return Time{Hour: t.Hour(), Minute: t.Minute()}, nil
}

func (t Time) String() string { return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute) }

// Before reports whether t is earlier in the day than u.
func (t Time) Before(u Time) bool {
	return t.Hour < u.Hour || (t.Hour == u.Hour && t.Minute < u.Minute)
}

// MarshalText writes the time as "15:04", so it can be used in JSON.
func (t Time) MarshalText() ([]byte, error) { return []byte(t.String()), nil }

// UnmarshalText reads a time written as "15:04".
func (t *Time) UnmarshalText(b []byte) error {
	parsed, err := Parse(string(b))
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

// Date is a calendar date.
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// ParseDate reads a date such as "2026-11-10".
func ParseDate(s string) (Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return Date{}, fmt.Errorf("%q is not a date such as \"2026-11-10\"", s)
	}
	return Date{Year: t.Year(), Month: t.Month(), Day: t.Day()}, nil
}

// DateOf returns the calendar date of t in t's own location.
func DateOf(t time.Time) Date {
	return Date{Year: t.Year(), Month: t.Month(), Day: t.Day()}
}

func (d Date) String() string { return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day) }

// On returns the instant at which the clock in loc reads t on date d.
//
// When clocks go forward a time can be skipped, and when they go back a time
// happens twice. On returns an error in both cases rather than guess.
func On(d Date, t Time, loc *time.Location) (time.Time, error) {
	guess := time.Date(d.Year, d.Month, d.Day, t.Hour, t.Minute, 0, 0, loc)
	_, before := guess.Add(-24 * time.Hour).Zone()
	_, after := guess.Add(24 * time.Hour).Zone()

	var found []time.Time
	for _, offset := range []int{before, after} {
		c := time.Date(d.Year, d.Month, d.Day, t.Hour, t.Minute, 0, 0, time.FixedZone("", offset)).In(loc)
		if DateOf(c) != d || c.Hour() != t.Hour || c.Minute() != t.Minute {
			continue
		}
		if len(found) == 0 || !found[0].Equal(c) {
			found = append(found, c)
		}
	}
	switch len(found) {
	case 0:
		return time.Time{}, fmt.Errorf("%s on %s does not exist in %s: the clocks go forward past it", t, d, loc)
	case 1:
		return found[0], nil
	}
	return time.Time{}, fmt.Errorf("%s on %s happens twice in %s: the clocks go back over it", t, d, loc)
}
