package cmd

import (
	"flag"

	"github.com/fatih/color"
	"github.com/mitchellh/cli"
	"github.com/paultyng/tfprovlint/provparse"
)

type schemaCommand struct {
	UI cli.Ui
}

func (c *schemaCommand) Help() string {
	return "help!"
}

func (c *schemaCommand) Synopsis() string {
	return "synopsis!"
}

func (c *schemaCommand) Run(args []string) int {
	flags := flag.NewFlagSet("lint", flag.ContinueOnError)
	err := flags.Parse(args)
	if err != nil {
		c.UI.Error(err.Error())
		return -1
	}

	prov, _, err := parseProvider(flags.Args())
	if err != nil {
		c.UI.Error(err.Error())
		return -1
	}

	if len(prov.DataSources) > 0 {
		c.UI.Output("Data Sources:")
		c.outputResources(prov.Resources)
	}

	if len(prov.Resources) > 0 {
		c.UI.Output("Resources:")
		c.outputResources(prov.Resources)
	}

	return 0
}

func (c *schemaCommand) outputResources(resources []provparse.Resource) {
	for _, r := range resources {
		c.UI.Output("\t" + color.WhiteString(r.Name))
		for _, att := range r.Attributes {
			c.outputAttribute(att, "\t\t")
		}
	}
}
func (c *schemaCommand) outputAttribute(att provparse.Attribute, prefix string) {
	c.UI.Output(prefix + color.WhiteString(att.Name))
	if len(att.Attributes) > 0 {
		prefix += "\t"
		for _, child := range att.Attributes {
			c.outputAttribute(child, prefix)
		}
	}
}

func SchemaCommandFactory(ui cli.Ui) cli.CommandFactory {
	return func() (cli.Command, error) {
		return &schemaCommand{
			UI: ui,
		}, nil
	}
}
