// Command agentlint resolves what an agent's configuration declares against
// what is actually on the machine.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/quietmachineworks/agentlint/internal/check"
	"github.com/quietmachineworks/agentlint/internal/inventory"
	"github.com/quietmachineworks/agentlint/internal/report"
)

// version is set at build time by the release tooling.
var version = "dev"

func main() {
	root := flag.String("config", "", "configuration directory to read (default: CLAUDE_CONFIG_DIR, else ~/.claude)")
	project := flag.String("project", ".", "repository whose .claude settings take part (empty to read none)")
	asJSON := flag.Bool("json", false, "write the run as JSON")
	strict := flag.Bool("strict", false, "exit non-zero on warnings too")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("agentlint", version)
		return
	}

	directory := *root
	if directory == "" {
		directory = inventory.DefaultRoot()
	}
	if _, err := os.Stat(directory); err != nil {
		fmt.Fprintf(os.Stderr, "agentlint: cannot read %s: %v\n", directory, err)
		os.Exit(2)
	}

	inv, err := inventory.Load(directory, *project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentlint: %v\n", err)
		os.Exit(2)
	}

	run := report.Summarise(inv, check.Run(inv, check.All()))

	if *asJSON {
		err = report.JSON(os.Stdout, run)
	} else {
		err = report.Text(os.Stdout, run)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentlint: %v\n", err)
		os.Exit(2)
	}

	if run.Counts.Errors > 0 || (*strict && run.Counts.Warnings > 0) {
		os.Exit(1)
	}
}
