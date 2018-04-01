package main

import (
	"log"
	"os"

	"github.com/paultyng/tfprovlint/provparse"
)

func main() {
	path := "$GOPATH/src/github.com/terraform-providers/terraform-provider-azurerm/azurerm"
	path = os.ExpandEnv(path)

	_, err := provparse.Package(path)
	if err != nil {
		log.Fatal(err)
	}
}
