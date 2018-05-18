package cmd

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/kisielk/gotool"
	"github.com/mitchellh/cli"

	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
)

type lintCommand struct {
	UI cli.Ui
}

type stringSliceFlags []string

func (i *stringSliceFlags) String() string {
	return "my string representation"
}

func (i *stringSliceFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (c *lintCommand) Help() string {
	return "help!"
}

func (c *lintCommand) Synopsis() string {
	return "synopsis!"
}

func (c *lintCommand) Run(args []string) int {
	var resourceNames stringSliceFlags
	var dataSourceNames stringSliceFlags
	var includeRules stringSliceFlags
	var excludeRules stringSliceFlags

	flags := flag.NewFlagSet("lint", flag.ContinueOnError)
	flags.Var(&includeRules, "include", "list of rules to include")
	flags.Var(&excludeRules, "exclude", "list of rules to exclude")
	flags.Var(&resourceNames, "rs", "list of resources to lint")
	flags.Var(&dataSourceNames, "ds", "list of data sources to lint")

	err := flags.Parse(args)
	if err != nil {
		c.UI.Error(err.Error())
		return -1
	}

	filtered := len(resourceNames) > 0 || len(dataSourceNames) > 0

	prov, basePath, err := parseProvider(flags.Args())
	if err != nil {
		c.UI.Error(err.Error())
		return -1
	}

	rules := loadRules(includeRules, excludeRules)
	results := []issueResult{}

	dataSources := prov.DataSources
	if filtered {
		dataSources = filterResources(dataSources, dataSourceNames)
	}
	newResults, err := evaluateRules(true, rules, dataSources)
	if err != nil {
		c.UI.Error(err.Error())
		return -1
	}
	results = append(results, newResults...)

	resources := prov.Resources
	if filtered {
		resources = filterResources(resources, resourceNames)
	}
	newResults, err = evaluateRules(false, rules, resources)
	if err != nil {
		c.UI.Error(err.Error())
		return -1
	}
	results = append(results, newResults...)

	c.UI.Output("")
	for _, res := range results {
		prefix := ""
		if res.ReadOnly {
			prefix = "data."
		}

		pos := prov.Fset.Position(res.Issue.Pos)
		fname := pos.Filename
		fname, err = filepath.Rel(basePath, fname)
		if err != nil {
			c.UI.Error(err.Error())
			return -1
		}
		spos := fmt.Sprintf("%s:%d:%d", fname, pos.Line, pos.Column)

		line := "[" + color.WhiteString("%s%s", prefix, res.Resource.Name) + "] " +
			"[" + color.RedString("%s", res.RuleID) + "] " +
			fmt.Sprintf("%s: ", spos) +
			color.WhiteString("%s", res.Issue.Message)

		c.UI.Output(line)
	}

	c.UI.Output(fmt.Sprintf("\n%d issues found", len(results)))

	return 0
}

func LintCommandFactory(ui cli.Ui) cli.CommandFactory {
	return func() (cli.Command, error) {
		return &lintCommand{
			UI: ui,
		}, nil
	}
}

type issueResult struct {
	ReadOnly bool
	Resource provparse.Resource
	RuleID   string
	Issue    lint.Issue
}

func parseProvider(paths []string) (*provparse.Provider, string, error) {
	paths = gotool.ImportPaths(paths)

	if len(paths) != 1 {
		log.Println("you must specify only one import path to lint")
	}

	return provparse.Package(paths[0])
}

func filterResources(resources []provparse.Resource, resourceNames []string) []provparse.Resource {
	if len(resourceNames) == 0 {
		return nil
	}

	nameMap := map[string]bool{}
	for _, n := range resourceNames {
		nameMap[n] = true
	}

	filtered := make([]provparse.Resource, 0, len(resourceNames))
	for _, r := range resources {
		if nameMap[r.Name] {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

func evaluateRules(readOnly bool, rules map[string]ruleFactoryFunc, resources []provparse.Resource) ([]issueResult, error) {
	results := []issueResult{}
	for _, r := range resources {
		for id, factory := range rules {
			rule := factory()
			newIssues, err := rule.CheckResource(readOnly, &r)
			if err != nil {
				return nil, err
			}
			for _, iss := range newIssues {
				results = append(results, issueResult{
					ReadOnly: readOnly,
					Issue:    iss,
					Resource: r,
					RuleID:   id,
				})
			}

		}
	}

	return results, nil
}
