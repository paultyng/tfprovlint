package main

import (
	"os"

	"github.com/mitchellh/cli"
	"github.com/paultyng/tfprovlint/cmd"
)

func main() {
	ui := &cli.ColoredUi{
		ErrorColor: cli.UiColorRed,
		Ui: &cli.BasicUi{
			ErrorWriter: os.Stderr,
			Reader:      os.Stdin,
			Writer:      os.Stdout,
		},
	}

	c := cli.NewCLI("tfprovlint", "1.0.0")

	c.Args = os.Args[1:]

	lintFact := cmd.LintCommandFactory(ui)

	c.Commands = map[string]cli.CommandFactory{
		"":       lintFact, // this no longer crashes but also not matched
		"lint":   lintFact,
		"schema": cmd.SchemaCommandFactory(ui),
	}

	exitStatus, err := c.Run()
	if err != nil {
		ui.Error(err.Error())
	}

	os.Exit(exitStatus)
}
