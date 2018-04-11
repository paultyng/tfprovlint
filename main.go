package main

import (
	"fmt"
	"log"

	"github.com/kisielk/gotool"
	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
	"github.com/paultyng/tfprovlint/rules"
)

func main() {
	// TODO: get this from flag or args?
	//path := "github.com/terraform-providers/terraform-provider-azurerm/azurerm"
	path := "github.com/terraform-providers/terraform-provider-aws/aws"
	//path := "github.com/terraform-providers/terraform-provider-template/template"

	paths := gotool.ImportPaths([]string{path})

	if len(paths) != 1 {
		log.Fatal("you must specify only one import path to lint")
	}

	prov, err := provparse.Package(paths[0])
	if err != nil {
		log.Fatal(err)
	}

	resourceRules := []lint.ResourceRule{
		//rules.NewMustSetProperTypes(),
		rules.NewCallBlacklist(),
	}

	issues := []lint.Issue{}
	for _, ds := range prov.Resources {
		for _, rule := range resourceRules {
			newIssues, err := rule.CheckResource(&ds)
			if err != nil {
				log.Fatal(err)
			}
			issues = append(issues, newIssues...)
		}
	}

	fmt.Println()
	for _, iss := range issues {
		fmt.Printf("%s: %s [%s]\n", prov.Fset.Position(iss.Pos), iss.Message, iss.RuleID)
	}
}
