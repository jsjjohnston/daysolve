// Package config holds the user's settings: working hours, breaks, buffer,
// time zone and the default input files. Settings live in one JSON file.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jsjjohnston/daysolve/internal/clock"
	"github.com/jsjjohnston/daysolve/solver"
)

// Config is the content of the config file.
type Config struct {
	// Zone is an IANA zone name such as "Europe/London", or "Local" for the
	// zone of the machine.
	Zone      string     `json:"zone"`
	WorkStart clock.Time `json:"work_start"`
	WorkEnd   clock.Time `json:"work_end"`
	Breaks    []Break    `json:"breaks"`
	Buffer    Duration   `json:"buffer"`

	// Tasks and Calendar are the files `plan` reads when no flag names one.
	// An empty Calendar means no calendar.
	Tasks    string `json:"tasks"`
	Calendar string `json:"calendar"`
}

// Break is a daily break between two times of day.
type Break struct {
	Start clock.Time `json:"start"`
	End   clock.Time `json:"end"`
}

// Duration is a time.Duration written in JSON as a string such as "10m".
type Duration time.Duration

func (d Duration) String() string {
	s := time.Duration(d).String()
	if strings.HasSuffix(s, "m0s") {
		s = strings.TrimSuffix(s, "0s")
	}
	if strings.HasSuffix(s, "h0m") {
		s = strings.TrimSuffix(s, "0m")
	}
	return s
}

func (d Duration) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

func (d *Duration) UnmarshalText(b []byte) error {
	parsed, err := time.ParseDuration(string(b))
	if err != nil {
		return fmt.Errorf("%q is not a duration such as \"10m\"", b)
	}
	*d = Duration(parsed)
	return nil
}

// Default returns the settings used when there is no config file: 09:00 to
// 17:00 in the machine's zone, lunch from 12:00 to 13:00, no buffer, tasks in
// tasks.json and no calendar.
func Default() Config {
	return Config{
		Zone:      "Local",
		WorkStart: clock.Time{Hour: 9},
		WorkEnd:   clock.Time{Hour: 17},
		Breaks:    []Break{{Start: clock.Time{Hour: 12}, End: clock.Time{Hour: 13}}},
		Tasks:     "tasks.json",
	}
}

// DefaultPath is the config file used when none is named:
// daysolve/config.json in the user's config directory.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find the config directory: %w", err)
	}
	return filepath.Join(dir, "daysolve", "config.json"), nil
}

// Load reads the config file. A missing file gives the defaults, and a key
// missing from the file keeps its default. An unknown key is an error.
func Load(path string) (Config, error) {
	c := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return Config{}, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

// Save writes the config file, creating its directory if needed.
func (c Config) Save(path string) error {
	data, err := c.JSON()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// JSON returns the config as indented JSON ending in a newline.
func (c Config) JSON() ([]byte, error) {
	if c.Breaks == nil {
		c.Breaks = []Break{}
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// Validate reports the first setting that cannot be used.
func (c Config) Validate() error {
	if _, err := c.Location(); err != nil {
		return err
	}
	if !c.WorkStart.Before(c.WorkEnd) {
		return fmt.Errorf("work_end (%s) must be after work_start (%s)", c.WorkEnd, c.WorkStart)
	}
	for _, b := range c.Breaks {
		if !b.Start.Before(b.End) {
			return fmt.Errorf("break %s-%s must end after it starts", b.Start, b.End)
		}
	}
	if c.Buffer < 0 {
		return errors.New("buffer must not be negative")
	}
	return nil
}

// Location returns the time zone named by Zone.
func (c Config) Location() (*time.Location, error) {
	if c.Zone == "" {
		return nil, errors.New("zone must not be empty; use an IANA name such as \"Europe/London\", or \"Local\"")
	}
	loc, err := time.LoadLocation(c.Zone)
	if err != nil {
		return nil, fmt.Errorf("zone %q is not a known time zone", c.Zone)
	}
	return loc, nil
}

// Keys lists the names Set accepts, in file order.
func Keys() []string {
	return []string{"zone", "work_start", "work_end", "breaks", "buffer", "tasks", "calendar"}
}

// Set changes one setting from its text form and checks the result is still
// valid. Breaks are written "12:00-13:00,15:00-15:15"; an empty value means
// no breaks.
func (c *Config) Set(key, value string) error {
	updated := *c
	var err error
	switch key {
	case "zone":
		updated.Zone = value
	case "work_start":
		updated.WorkStart, err = clock.Parse(value)
	case "work_end":
		updated.WorkEnd, err = clock.Parse(value)
	case "breaks":
		updated.Breaks, err = parseBreaks(value)
	case "buffer":
		err = updated.Buffer.UnmarshalText([]byte(value))
	case "tasks":
		if value == "" {
			err = errors.New("must not be empty")
		}
		updated.Tasks = value
	case "calendar":
		updated.Calendar = value
	default:
		return fmt.Errorf("unknown setting %q; settings are: %s", key, strings.Join(Keys(), ", "))
	}
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	if err := updated.Validate(); err != nil {
		return err
	}
	*c = updated
	return nil
}

func parseBreaks(value string) ([]Break, error) {
	breaks := []Break{}
	if strings.TrimSpace(value) == "" {
		return breaks, nil
	}
	for _, part := range strings.Split(value, ",") {
		from, to, ok := strings.Cut(strings.TrimSpace(part), "-")
		if !ok {
			return nil, fmt.Errorf("%q is not a break such as \"12:00-13:00\"", part)
		}
		start, err := clock.Parse(from)
		if err != nil {
			return nil, err
		}
		end, err := clock.Parse(to)
		if err != nil {
			return nil, err
		}
		breaks = append(breaks, Break{Start: start, End: end})
	}
	return breaks, nil
}

// Day returns the working day for a date: working hours, breaks and buffer
// as instants in the configured zone.
func (c Config) Day(d clock.Date) (solver.Day, error) {
	loc, err := c.Location()
	if err != nil {
		return solver.Day{}, err
	}
	start, err := clock.On(d, c.WorkStart, loc)
	if err != nil {
		return solver.Day{}, fmt.Errorf("work_start: %w", err)
	}
	end, err := clock.On(d, c.WorkEnd, loc)
	if err != nil {
		return solver.Day{}, fmt.Errorf("work_end: %w", err)
	}
	day := solver.Day{Start: start, End: end, Buffer: time.Duration(c.Buffer)}
	for _, b := range c.Breaks {
		bs, err := clock.On(d, b.Start, loc)
		if err != nil {
			return solver.Day{}, fmt.Errorf("break: %w", err)
		}
		be, err := clock.On(d, b.End, loc)
		if err != nil {
			return solver.Day{}, fmt.Errorf("break: %w", err)
		}
		day.Breaks = append(day.Breaks, solver.Interval{Start: bs, End: be})
	}
	return day, nil
}
