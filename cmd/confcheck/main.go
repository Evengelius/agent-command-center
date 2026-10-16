// Command confcheck is the configuration check of doc/reliability.md. It
// writes only to the path given with -out and never opens credential files.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/Evengelius/agent-command-center/internal/confcheck"
)

func main() { os.Exit(run(os.Args[1:])) }

// run returns 0 PASS, 1 FAIL, 2 NOT-VERIFIED for compare; 3 for usage or errors.
func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: confcheck snapshot -out f.json [-project dir] | compare -before f.json [-after f.json] [-project dir] [-cli-ran]")
		return 3
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 3
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	out := fs.String("out", "", "snapshot: baseline file to write")
	before := fs.String("before", "", "compare: baseline file")
	after := fs.String("after", "", "compare: second baseline (default: live now)")
	project := fs.String("project", "", "project directory (.claude and .codex files are checked)")
	cliRan := fs.Bool("cli-ran", false, "compare: a CLI ran during the window")
	if err := fs.Parse(args[1:]); err != nil {
		return 3
	}
	c := &confcheck.Checker{Home: home, Project: *project, Procs: confcheck.TasklistLister{}}

	switch args[0] {
	case "snapshot":
		if *out == "" {
			fmt.Fprintln(os.Stderr, "snapshot needs -out")
			return 3
		}
		s, err := c.Snapshot()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 3
		}
		data, _ := json.MarshalIndent(s, "", "  ")
		if err := os.WriteFile(*out, data, 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 3
		}
		fmt.Printf("snapshot: %d files, %d session files, claude running: %t, codex running: %t\n",
			len(s.Entries), len(s.Sessions), s.Claude, s.Codex)
		return 0
	case "compare":
		b, err := load(*before)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 3
		}
		if *project == "" {
			c.Project = b.Project
		}
		var a *confcheck.Snapshot
		if *after != "" {
			a, err = load(*after)
		} else {
			a, err = c.Snapshot()
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 3
		}
		r := confcheck.Compare(b, a, *cliRan)
		fmt.Print(r)
		return r.Verdict.ExitCode()
	}
	fmt.Fprintln(os.Stderr, "unknown command:", args[0])
	return 3
}

func load(path string) (*confcheck.Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s confcheck.Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
