package main

import (
	"fmt"
	"log"

	"github.com/kisielk/gotool"
	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
	"github.com/paultyng/tfprovlint/rules"
)

type ruleFactoryFunc func() lint.ResourceRule

var resourceRules = map[string]ruleFactoryFunc{
	"tfprovlint001": rules.NewNoSetIdInDeleteFuncRule,
	"tfprovlint002": rules.NewSetAttributeNameExistsRule,
	// "tfprovlint003": use proper types in set
	// "tfprovlint004": err check sets on complex types
	"tfprovlint005": rules.NewDoNotDereferencePointersInSet,
}

type issueResult struct {
	ResourceType string // "resource" or "data source"
	Resource     provparse.Resource
	RuleID       string
	Issue        lint.Issue
}

func main() {
	// TODO: get this from flag or args?
	//path := "github.com/terraform-providers/terraform-provider-azurerm/azurerm"
	//path := "github.com/terraform-providers/terraform-provider-aws/aws"
	path := "github.com/terraform-providers/terraform-provider-template/template"

	paths := gotool.ImportPaths([]string{path})

	if len(paths) != 1 {
		log.Fatal("you must specify only one import path to lint")
	}

	prov, err := provparse.Package(paths[0])
	if err != nil {
		log.Fatal(err)
	}

	results := []issueResult{}
	newResults, err := evaluateRules("data", resourceRules, prov.DataSources)
	if err != nil {
		log.Fatal(err)
	}
	results = append(results, newResults...)

	newResults, err = evaluateRules("resource", resourceRules, prov.Resources)
	if err != nil {
		log.Fatal(err)
	}
	results = append(results, newResults...)

	fmt.Println()
	for _, res := range results {
		fmt.Printf("%s: [%s] [%s.%s] %s\n", prov.Fset.Position(res.Issue.Pos), res.RuleID, res.ResourceType, res.Resource.Name, res.Issue.Message)
	}
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
