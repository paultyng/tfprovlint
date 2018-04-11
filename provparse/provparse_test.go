package provparse_test

import (
	"testing"

	"github.com/paultyng/tfprovlint/provparse"
)

func TestPackage_Template(t *testing.T) {
	// This assumes you have cloned the template provider to your $GOPATH
	path := "github.com/terraform-providers/terraform-provider-template/template"

	prov := parsePackage(t, path)

	dsTemplateFile := prov.DataSource("template_file")
	if dsTemplateFile == nil {
		t.Fatal("expected data source template_file")
	}
	if dsTemplateFile.ReadFunc == nil {
		t.Fatal("read func is required for template_file")
	}
	if dsTemplateFile.CreateFunc != nil ||
		dsTemplateFile.UpdateFunc != nil ||
		dsTemplateFile.DeleteFunc != nil {
		t.Fatal("create, update, and delete funcs must be nil for template_file")
	}
	if att := dsTemplateFile.Attribute("filename"); att != nil {
		if att.Type != provparse.TypeString {
			t.Fatal("template_file.filename is not a TypeString")
		}
	} else {
		t.Fatal("filename attribute was not found on template_file")
	}

	dsTemplateCloudinitConfig := prov.DataSource("template_cloudinit_config")
	if dsTemplateCloudinitConfig == nil {
		t.Fatal("expected data source template_cloudinit_config")
	}

	if dsTemplateCloudinitConfig.ReadFunc == nil {
		t.Fatal("read func is required for template_cloudinit_config")
	}

	if dsTemplateCloudinitConfig.CreateFunc != nil ||
		dsTemplateCloudinitConfig.UpdateFunc != nil ||
		dsTemplateCloudinitConfig.DeleteFunc != nil {
		t.Fatal("create, update, and delete funcs must be nil for template_cloudinit_config")
	}

	rTemplateDir := prov.Resource("template_dir")
	if rTemplateDir == nil {
		t.Fatal("expected resource template_dir")
	}

	if rTemplateDir.CreateFunc == nil ||
		rTemplateDir.ReadFunc == nil ||
		rTemplateDir.DeleteFunc == nil {
		t.Fatal("expect create, read, and delete funcs for template_dir")
	}
}

func parsePackage(t *testing.T, path string) *provparse.Provider {
	t.Helper()

	prov, err := provparse.Package(path)
	if err != nil {
		t.Fatalf("unable to parse package: %s", err)
	}
	if prov == nil {
		t.Fatal("no provider returned")
	}

	return prov
}
