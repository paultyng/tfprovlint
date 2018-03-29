package provparse_test

import (
	"testing"

	// importing this here just to ensure its present in vendor
	_ "github.com/terraform-providers/terraform-provider-template/template"

	"github.com/paultyng/tfprovlint/provparse"
)

func TestPackage(t *testing.T) {
	path := "../vendor/github.com/terraform-providers/terraform-provider-template/template"

	prov, err := provparse.Package(path)
	if err != nil {
		t.Fatal(err)
	}
	if prov == nil {
		t.Fatal("no provider returned")
	}

	dsTemplateFile := prov.DataSource("template_file")
	if dsTemplateFile == nil {
		t.Fatal("expected data source template_file")
	}

	dsTemplateCloudinitConfig := prov.DataSource("template_cloudinit_config")
	if dsTemplateCloudinitConfig == nil {
		t.Fatal("expected data source template_cloudinit_config")
	}

	rTemplateDir := prov.Resource("template_dir")
	if rTemplateDir == nil {
		t.Fatal("expected resource template_dir")
	}
}
