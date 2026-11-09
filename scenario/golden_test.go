package scenario

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/jsjjohnston/daysolve/solver"
)

var update = flag.Bool("update", false, "rewrite schedule.json in every scenario from the solver's current output")

const scenarioGlob = "../testdata/scenarios/*/problem.json"

// solveFile reads a problem file, solves it and returns the schedule as JSON.
func solveFile(t *testing.T, path string) []byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	p, err := ReadProblem(f)
	if err != nil {
		t.Fatal(err)
	}
	s, err := solver.Solve(p)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := WriteSchedule(&out, s); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// TestScenarios solves every testdata/scenarios/<name>/problem.json and
// compares the result with the schedule.json beside it. Run
// `go test ./scenario -update` to rewrite the expected schedules, then review
// the diff by hand.
func TestScenarios(t *testing.T) {
	problems, err := filepath.Glob(scenarioGlob)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) == 0 {
		t.Fatalf("no scenarios match %s", scenarioGlob)
	}
	for _, problem := range problems {
		dir := filepath.Dir(problem)
		t.Run(filepath.Base(dir), func(t *testing.T) {
			got := solveFile(t, problem)
			golden := filepath.Join(dir, "schedule.json")
			if *update {
				if err := os.WriteFile(golden, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("schedule differs from %s\ngot:\n%s\nwant:\n%s", golden, got, want)
			}
		})
	}
}

// TestScenariosAreDeterministic solves every scenario repeatedly and requires
// byte-identical output each time.
func TestScenariosAreDeterministic(t *testing.T) {
	problems, err := filepath.Glob(scenarioGlob)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) == 0 {
		t.Fatalf("no scenarios match %s", scenarioGlob)
	}
	for _, problem := range problems {
		t.Run(filepath.Base(filepath.Dir(problem)), func(t *testing.T) {
			first := solveFile(t, problem)
			for range 10 {
				if again := solveFile(t, problem); !bytes.Equal(first, again) {
					t.Fatalf("output changed between runs\nfirst:\n%s\nagain:\n%s", first, again)
				}
			}
		})
	}
}
