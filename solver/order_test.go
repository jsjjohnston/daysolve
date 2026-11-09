package solver

import (
	"slices"
	"testing"
)

func ids(tasks []Task) []string {
	out := []string{}
	for _, t := range tasks {
		out = append(out, t.ID)
	}
	return out
}

func TestOrdered(t *testing.T) {
	tests := []struct {
		name  string
		tasks []Task
		want  []string
	}{
		{"nothing", nil, []string{}},
		{
			"earliest deadline first",
			[]Task{{ID: "late", Deadline: at("16:00")}, {ID: "early", Deadline: at("11:00")}},
			[]string{"early", "late"},
		},
		{
			"a deadline beats no deadline",
			[]Task{{ID: "none", Priority: 1}, {ID: "due", Deadline: at("16:00")}},
			[]string{"due", "none"},
		},
		{
			"deadline beats priority",
			[]Task{{ID: "urgent", Priority: 1, Deadline: at("16:00")}, {ID: "sooner", Priority: 9, Deadline: at("11:00")}},
			[]string{"sooner", "urgent"},
		},
		{
			"lower priority number first",
			[]Task{{ID: "c", Priority: 3}, {ID: "a", Priority: 1}, {ID: "b", Priority: 2}},
			[]string{"a", "b", "c"},
		},
		{
			"a priority beats no priority",
			[]Task{{ID: "none"}, {ID: "low", Priority: 9}},
			[]string{"low", "none"},
		},
		{
			"ties keep input order",
			[]Task{{ID: "z", Priority: 2}, {ID: "m", Priority: 2}, {ID: "a", Priority: 2}},
			[]string{"z", "m", "a"},
		},
		{
			"same instant in another zone is a tie",
			[]Task{{ID: "first", Deadline: at("16:00").UTC()}, {ID: "second", Deadline: at("16:00")}},
			[]string{"first", "second"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ids(ordered(tt.tasks)); !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrderedLeavesInputAlone(t *testing.T) {
	in := []Task{{ID: "b", Priority: 2}, {ID: "a", Priority: 1}}
	ordered(in)
	if got, want := ids(in), []string{"b", "a"}; !slices.Equal(got, want) {
		t.Errorf("input changed to %v", got)
	}
}
