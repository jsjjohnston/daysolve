package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/jsjjohnston/daysolve/config"
)

// runConfig handles "daysolve config <show|set|init|path>".
func runConfig(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daysolve config", flag.ContinueOnError)
	fs.SetOutput(stderr)
	pathFlag := fs.String("config", "", "config `file` to use instead of the default")
	force := fs.Bool("force", false, "with init: replace an existing config file")

	operands, err := parseArgs(fs, args)
	if err != nil {
		return flagExit(err)
	}
	if len(operands) == 0 {
		fmt.Fprint(stderr, "daysolve: config needs a subcommand: show, set, init or path\n")
		return exitUsage
	}
	sub, operands := operands[0], operands[1:]

	path, err := configPath(*pathFlag)
	if err != nil {
		return fail(stderr, err)
	}

	wantOperands := map[string]int{"path": 0, "show": 0, "init": 0, "set": 2}
	n, known := wantOperands[sub]
	if !known {
		fmt.Fprintf(stderr, "daysolve: unknown config subcommand %q: use show, set, init or path\n", sub)
		return exitUsage
	}
	if len(operands) != n {
		if sub == "set" {
			fmt.Fprint(stderr, "daysolve: usage: daysolve config set <key> <value>\n")
		} else {
			fmt.Fprintf(stderr, "daysolve: config %s takes no arguments\n", sub)
		}
		return exitUsage
	}

	switch sub {
	case "path":
		fmt.Fprintln(stdout, path)
	case "show":
		c, err := config.Load(path)
		if err != nil {
			return fail(stderr, err)
		}
		data, err := c.JSON()
		if err != nil {
			return fail(stderr, err)
		}
		stdout.Write(data)
	case "init":
		if _, err := os.Stat(path); err == nil && !*force {
			return fail(stderr, fmt.Errorf("%s already exists; use --force to replace it", path))
		}
		if err := config.Default().Save(path); err != nil {
			return fail(stderr, err)
		}
		fmt.Fprintf(stdout, "Wrote %s\n", path)
	case "set":
		c, err := config.Load(path)
		if err != nil {
			return fail(stderr, err)
		}
		if err := c.Set(operands[0], operands[1]); err != nil {
			return fail(stderr, err)
		}
		if err := c.Save(path); err != nil {
			return fail(stderr, err)
		}
		fmt.Fprintf(stdout, "Set %s to %q in %s\n", operands[0], operands[1], path)
	}
	return exitOK
}

// flagExit maps a flag parsing error to an exit code. Asking for help is not
// an error.
func flagExit(err error) int {
	if errors.Is(err, flag.ErrHelp) {
		return exitOK
	}
	return exitUsage
}
