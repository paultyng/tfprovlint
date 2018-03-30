package main

import (
	"log"

	"github.com/paultyng/tfprovlint/provparse"
)

func main() {
	path := "/home/paul/go/src/github.com/terraform-providers/terraform-provider-azurerm/azurerm"

	_, err := provparse.Package(path)
	if err != nil {
		log.Fatal(err)
	}
}
