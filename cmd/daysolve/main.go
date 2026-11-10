// Command daysolve plans a day from a task file and a calendar.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jsjjohnston/daysolve/config"
)

const usage = `daysolve plans a day from a task file and a calendar.

Usage:
  daysolve plan [flags]              plan a day and print the schedule
  daysolve config show               print the settings in use
  daysolve config set <key> <value>  change one setting
  daysolve config init [--force]     write a config file holding the defaults
  daysolve config path               print where the config file is
  daysolve help                      print this message

Run "daysolve plan -h" for the plan flags.
Settings: zone, work_start, work_end, breaks, buffer, tasks, calendar.
Every command accepts --config <file>; DAYSOLVE_CONFIG names one too.
`

// Exit codes.
const (
	exitOK    = 0
	exitError = 1 // the command could not be carried out
	exitUsage = 2 // the command line was wrong
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, time.Now()))
}

// run carries out one command line. now is the current time; it is passed in
// so that tests control it.
func run(args []string, stdout, stderr io.Writer, now time.Time) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return exitUsage
	}
	switch args[0] {
	case "plan":
		return runPlan(args[1:], stdout, stderr, now)
	case "config":
		return runConfig(args[1:], stdout, stderr)
	case "help", "-h", "-help", "--help":
		fmt.Fprint(stdout, usage)
		return exitOK
	}
	fmt.Fprintf(stderr, "daysolve: unknown command %q\n\n%s", args[0], usage)
	return exitUsage
}

// fail prints an error and returns the exit code for it.
func fail(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "daysolve: %v\n", err)
	return exitError
}

// configPath picks the config file: the flag, then DAYSOLVE_CONFIG, then the
// default location.
func configPath(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if env := os.Getenv("DAYSOLVE_CONFIG"); env != "" {
		return env, nil
	}
	return config.DefaultPath()
}

// parseArgs parses flags that may come before, between or after the other
// arguments, and returns the other arguments in order.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var operands []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return operands, nil
		}
		operands = append(operands, fs.Arg(0))
		args = fs.Args()[1:]
	}
}
