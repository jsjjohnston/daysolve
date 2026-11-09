package solver

import (
	"slices"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   []Interval
		want []string
	}{
		{"nothing", nil, []string{}},
		{"sorts", []Interval{span("11:00", "12:00"), span("09:00", "10:00")}, []string{"09:00-10:00", "11:00-12:00"}},
		{"merges overlap", []Interval{span("09:00", "10:30"), span("10:00", "11:00")}, []string{"09:00-11:00"}},
		{"merges touching", []Interval{span("09:00", "10:00"), span("10:00", "11:00")}, []string{"09:00-11:00"}},
		{"merges contained", []Interval{span("09:00", "12:00"), span("10:00", "11:00")}, []string{"09:00-12:00"}},
		{"drops empty", []Interval{span("09:00", "09:00"), span("10:00", "11:00")}, []string{"10:00-11:00"}},
		{"drops reversed", []Interval{span("12:00", "11:00")}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clocks(normalize(tt.in)); !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeLeavesInputAlone(t *testing.T) {
	in := []Interval{span("11:00", "12:00"), span("09:00", "10:00")}
	normalize(in)
	if got, want := clocks(in), []string{"11:00-12:00", "09:00-10:00"}; !slices.Equal(got, want) {
		t.Errorf("input changed to %v", got)
	}
}

func TestSubtract(t *testing.T) {
	day := []Interval{span("09:00", "17:00")}
	tests := []struct {
		name string
		base []Interval
		cut  []Interval
		want []string
	}{
		{"nothing to cut", day, nil, []string{"09:00-17:00"}},
		{"cut in the middle", day, []Interval{span("12:00", "13:00")}, []string{"09:00-12:00", "13:00-17:00"}},
		{"cut at the start", day, []Interval{span("09:00", "10:00")}, []string{"10:00-17:00"}},
		{"cut at the end", day, []Interval{span("16:00", "17:00")}, []string{"09:00-16:00"}},
		{"cut overhangs both ends", day, []Interval{span("08:00", "18:00")}, []string{}},
		{"cut overhangs the start", day, []Interval{span("08:00", "10:00")}, []string{"10:00-17:00"}},
		{"cut overhangs the end", day, []Interval{span("16:00", "18:00")}, []string{"09:00-16:00"}},
		{"cut outside", day, []Interval{span("07:00", "08:00"), span("18:00", "19:00")}, []string{"09:00-17:00"}},
		{"cut touching the outside", day, []Interval{span("08:00", "09:00"), span("17:00", "18:00")}, []string{"09:00-17:00"}},
		{"several cuts", day, []Interval{span("10:00", "11:00"), span("14:00", "15:00")}, []string{"09:00-10:00", "11:00-14:00", "15:00-17:00"}},
		{
			"one cut across two bases",
			[]Interval{span("09:00", "11:00"), span("12:00", "14:00")},
			[]Interval{span("10:00", "13:00")},
			[]string{"09:00-10:00", "13:00-14:00"},
		},
		{"empty base", nil, []Interval{span("10:00", "11:00")}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clocks(subtract(tt.base, tt.cut)); !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
