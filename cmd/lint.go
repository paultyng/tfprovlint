package cmd

import (
	"flag"
	"fmt"
	"log"

	"github.com/fatih/color"
	"github.com/kisielk/gotool"
	"github.com/mitchellh/cli"

	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
	"github.com/paultyng/tfprovlint/rules"
)

type lintCommand struct {
	UI cli.Ui
}

func (c *lintCommand) Help() string {
	return "help!"
}

func (c *lintCommand) Synopsis() string {
	return "synopsis!"
}

func (c *lintCommand) Run(args []string) int {
	flags := flag.NewFlagSet("lint", flag.ContinueOnError)
	err := flags.Parse(args)
	if err != nil {
		c.UI.Error(err.Error())
		return -1
	}

	paths := flags.Args()
	paths = gotool.ImportPaths(paths)

	if len(paths) != 1 {
		log.Println("you must specify only one import path to lint")
	}

	prov, err := provparse.Package(paths[0])
	if err != nil {
		c.UI.Error(err.Error())
		return -1
	}

	results := []issueResult{}
	newResults, err := evaluateRules("data", resourceRules, prov.DataSources)
	if err != nil {
		c.UI.Error(err.Error())
		return -1
	}
	results = append(results, newResults...)

	newResults, err = evaluateRules("resource", resourceRules, prov.Resources)
	if err != nil {
		c.UI.Error(err.Error())
		return -1
	}
	results = append(results, newResults...)

	c.UI.Output("")
	for _, res := range results {
		line := "[" + color.WhiteString("%s.%s", res.ResourceType, res.Resource.Name) + "] " +
			"[" + color.RedString("%s", res.RuleID) + "] " +
			fmt.Sprintf("%s: ", prov.Fset.Position(res.Issue.Pos)) +
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

type ruleFactoryFunc func() lint.ResourceRule

var resourceRules = map[string]ruleFactoryFunc{
	"tfprovlint001": rules.NewNoSetIdInDeleteFuncRule,
	"tfprovlint002": rules.NewSetAttributeNameExistsRule,
	"tfprovlint003": rules.NewUseProperAttributeTypesInSetRule,
	// "tfprovlint004": err check sets on complex types
	"tfprovlint005": rules.NewDoNotDereferencePointersInSetRule,
}

type issueResult struct {
	ResourceType string // "resource" or "data source"
	Resource     provparse.Resource
	RuleID       string
	Issue        lint.Issue
}

func evaluateRules(resourceType string, rules map[string]ruleFactoryFunc, resources []provparse.Resource) ([]issueResult, error) {
	results := []issueResult{}
	for _, r := range resources {
		// if r.Name == "aws_ssm_maintenance_window_task" {
		// 	r.ReadFunc.WriteTo(os.Stdout)
		// }

		for id, factory := range resourceRules {
			rule := factory()
			newIssues, err := rule.CheckResource(&r)
			if err != nil {
				return nil, err
			}
			for _, iss := range newIssues {
				results = append(results, issueResult{
					ResourceType: resourceType,
					Issue:        iss,
					Resource:     r,
					RuleID:       id,
				})
			}

		}
	}

	return results, nil
}
