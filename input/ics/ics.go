// Package ics reads fixed events from an iCalendar (.ics) file using only the
// standard library.
//
// It reads VEVENTs with UTC, TZID or floating times. It leaves out all-day,
// cancelled and transparent ("free") events, because they do not block time.
// It does not expand recurring events: those are returned with Recurring set
// so the caller can refuse them rather than plan around a wrong calendar.
package ics

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Event is one timed event from the calendar.
type Event struct {
	Title string
	Start time.Time
	End   time.Time

	// Recurring is true when the event repeats (RRULE or RDATE). Start and
	// End then describe only the first occurrence.
	Recurring bool
}

type property struct {
	params map[string]string
	value  string
}

type rawEvent struct {
	summary   string
	start     *property
	end       *property
	duration  string
	status    string
	transp    string
	recurring bool
}

// Read returns the timed events in the calendar, in start order, with every
// time converted to loc. A floating time (one with no zone) is read as a
// time in loc.
func Read(r io.Reader, loc *time.Location) ([]Event, error) {
	lines, err := unfold(r)
	if err != nil {
		return nil, err
	}

	events := []Event{}
	var cur *rawEvent
	nested := 0 // depth of components inside the current event, such as VALARM
	sawCalendar := false
	for _, line := range lines {
		name, params, value, ok := parseLine(line)
		if !ok {
			continue
		}
		component := strings.ToUpper(strings.TrimSpace(value))
		switch {
		case cur == nil:
			if name == "BEGIN" && component == "VCALENDAR" {
				sawCalendar = true
			}
			if name == "BEGIN" && component == "VEVENT" {
				cur = &rawEvent{}
			}
		case name == "BEGIN":
			nested++
		case name == "END" && nested > 0:
			nested--
		case name == "END" && component == "VEVENT":
			event, keep, err := cur.event(loc)
			if err != nil {
				return nil, fmt.Errorf("event %q: %w", cur.title(), err)
			}
			if keep {
				events = append(events, event)
			}
			cur = nil
		case nested == 0:
			cur.set(name, params, value)
		}
	}
	if !sawCalendar {
		return nil, errors.New("not an iCalendar file: no BEGIN:VCALENDAR line")
	}
	if cur != nil {
		return nil, fmt.Errorf("event %q: no END:VEVENT line", cur.title())
	}
	slices.SortStableFunc(events, func(a, b Event) int { return a.Start.Compare(b.Start) })
	return events, nil
}

// unfold reads content lines, joining continuation lines (those that start
// with a space or tab) onto the line before.
func unfold(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if len(lines) > 0 && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) {
			lines[len(lines)-1] += line[1:]
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read calendar: %w", err)
	}
	return lines, nil
}

// parseLine splits "NAME;PARAM=value:content" into its parts. Parameter
// values may be quoted and may then contain colons and semicolons.
func parseLine(line string) (name string, params map[string]string, value string, ok bool) {
	colon := indexOutsideQuotes(line, ':')
	if colon < 0 {
		return "", nil, "", false
	}
	head, value := line[:colon], line[colon+1:]
	params = map[string]string{}
	for i := 0; ; i++ {
		semi := indexOutsideQuotes(head, ';')
		part := head
		if semi >= 0 {
			part = head[:semi]
		}
		if i == 0 {
			name = strings.ToUpper(strings.TrimSpace(part))
		} else if k, v, found := strings.Cut(part, "="); found {
			params[strings.ToUpper(strings.TrimSpace(k))] = strings.Trim(v, `"`)
		}
		if semi < 0 {
			break
		}
		head = head[semi+1:]
	}
	return name, params, value, true
}

func indexOutsideQuotes(s string, target byte) int {
	quoted := false
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '"':
			quoted = !quoted
		case s[i] == target && !quoted:
			return i
		}
	}
	return -1
}

var unescape = strings.NewReplacer(`\\`, `\`, `\n`, " ", `\N`, " ", `\,`, ",", `\;`, ";")

func (e *rawEvent) set(name string, params map[string]string, value string) {
	switch name {
	case "SUMMARY":
		e.summary = strings.TrimSpace(unescape.Replace(value))
	case "DTSTART":
		e.start = &property{params: params, value: strings.TrimSpace(value)}
	case "DTEND":
		e.end = &property{params: params, value: strings.TrimSpace(value)}
	case "DURATION":
		e.duration = strings.TrimSpace(value)
	case "STATUS":
		e.status = strings.ToUpper(strings.TrimSpace(value))
	case "TRANSP":
		e.transp = strings.ToUpper(strings.TrimSpace(value))
	case "RRULE", "RDATE":
		e.recurring = true
	}
}

func (e *rawEvent) title() string {
	if e.summary == "" {
		return "(untitled)"
	}
	return e.summary
}

// event converts the raw properties. keep is false for an event that does
// not block time.
func (e *rawEvent) event(loc *time.Location) (event Event, keep bool, err error) {
	if e.status == "CANCELLED" || e.transp == "TRANSPARENT" {
		return Event{}, false, nil
	}
	if e.start == nil {
		return Event{}, false, errors.New("no DTSTART")
	}
	if e.start.isDate() {
		return Event{}, false, nil // all-day
	}
	start, err := e.start.time(loc)
	if err != nil {
		return Event{}, false, fmt.Errorf("DTSTART: %w", err)
	}
	end := start
	switch {
	case e.end != nil:
		if end, err = e.end.time(loc); err != nil {
			return Event{}, false, fmt.Errorf("DTEND: %w", err)
		}
	case e.duration != "":
		d, err := parseDuration(e.duration)
		if err != nil {
			return Event{}, false, fmt.Errorf("DURATION: %w", err)
		}
		end = start.Add(d)
	}
	if end.Before(start) {
		return Event{}, false, errors.New("ends before it starts")
	}
	return Event{Title: e.title(), Start: start, End: end, Recurring: e.recurring}, true, nil
}

// isDate reports whether the property is a date with no time of day.
func (p *property) isDate() bool {
	return strings.EqualFold(p.params["VALUE"], "DATE") || len(p.value) == len("20060102")
}

// time parses a date-time: UTC ("...Z"), in a named zone (TZID), or floating.
func (p *property) time(loc *time.Location) (time.Time, error) {
	if strings.HasSuffix(p.value, "Z") {
		t, err := time.Parse("20060102T150405Z", p.value)
		if err != nil {
			return time.Time{}, fmt.Errorf("%q is not a date-time", p.value)
		}
		return t.In(loc), nil
	}
	zone := loc
	if tzid := p.params["TZID"]; tzid != "" {
		named, err := time.LoadLocation(tzid)
		if err != nil {
			return time.Time{}, fmt.Errorf("time zone %q is not an IANA zone name", tzid)
		}
		zone = named
	}
	t, err := time.ParseInLocation("20060102T150405", p.value, zone)
	if err != nil {
		return time.Time{}, fmt.Errorf("%q is not a date-time", p.value)
	}
	return t.In(loc), nil
}

var durationPattern = regexp.MustCompile(`^\+?P(?:(\d+)W)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?$`)

// parseDuration reads an iCalendar duration such as "PT1H30M" or "P1D".
func parseDuration(s string) (time.Duration, error) {
	m := durationPattern.FindStringSubmatch(strings.ToUpper(s))
	if m == nil {
		return 0, fmt.Errorf("%q is not a duration such as \"PT1H30M\"", s)
	}
	units := []time.Duration{7 * 24 * time.Hour, 24 * time.Hour, time.Hour, time.Minute, time.Second}
	var total time.Duration
	for i, unit := range units {
		if m[i+1] == "" {
			continue
		}
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return 0, fmt.Errorf("%q is not a duration such as \"PT1H30M\"", s)
		}
		total += time.Duration(n) * unit
	}
	return total, nil
}
