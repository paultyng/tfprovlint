package rules

import (
	"fmt"
	"testing"

	"github.com/paultyng/tfprovlint/provparse"
)

func TestUseProperAttributeTypesInSet(t *testing.T) {
	for i, c := range []struct {
		expectedMsg   string
		attributeType provparse.AttributeType
		funcName      string
	}{
		{"", provparse.TypeBool, "setBool"},

		{"", provparse.TypeInt, "setInt"},
		{"", provparse.TypeInt, "setInt16"},
		{"", provparse.TypeInt, "setInt32"},
		{"", provparse.TypeInt, "setInt64"},
		{"", provparse.TypeInt, "setInt8"},

		{"", provparse.TypeFloat, "setFloat32"},
		{"", provparse.TypeFloat, "setFloat64"},

		{"", provparse.TypeString, "setString"},
		{"", provparse.TypeString, "setPointerString"},

		{"", provparse.TypeString, "setStructFieldString"},
		{"", provparse.TypeString, "setStructFieldPointerString"},
		{"", provparse.TypeString, "setPointerStructFieldString"},
		{"", provparse.TypeString, "setPointerStructFieldPointerString"},

		{"", provparse.TypeString, "setNamedString"},
		{"", provparse.TypeInt, "setNamedInt"},

		{"attribute \"att\" expects a d.Set compatible with TypeBool", provparse.TypeBool, "setInt"},
	} {
		t.Run(fmt.Sprintf("%d %s", i, c.funcName), func(t *testing.T) {
			ci := lookupSetCallInstruction(useProperAttributeTypesPkg, c.funcName)
			if ci == nil {
				t.Fatalf("unable to find ssa.CallInstruction %q", c.funcName)
			}
			att := &provparse.Attribute{
				Name: "att",
				Type: c.attributeType,
			}
			actualIssues, err := useProperAttributeTypesInSet(nil, att, "att", ci)
			if err != nil {
				t.Fatal(err)
			}
			assertIssueMsg(t, c.expectedMsg, actualIssues)
		})
	}
}

var useProperAttributeTypesPkg = mustMakeSamplePkg(`
package test

type ResourceData struct {}

func (rd *ResourceData) Set(string, interface{}) error {
	// no-op
	return nil
}

func setBool(d *ResourceData, val bool) {
	d.Set("att", val)
}

func setInt(d *ResourceData, val int) {
	d.Set("att", val)
}

func setInt16(d *ResourceData, val int16) {
	d.Set("att", val)
}

func setInt32(d *ResourceData, val int32) {
	d.Set("att", val)
}

func setInt64(d *ResourceData, val int64) {
	d.Set("att", val)
}

func setInt8(d *ResourceData, val int8) {
	d.Set("att", val)
}

func setFloat32(d *ResourceData, val float32) {
	d.Set("att", val)
}

func setFloat64(d *ResourceData, val float64) {
	d.Set("att", val)
}

func setString(d *ResourceData, val string) {
	d.Set("att", val)
}

func setPointerString(d *ResourceData, val *string) {
	d.Set("att", val)
}

type MyStruct struct {
	String string
	PointerString *string
}

func setStructFieldString(d *ResourceData, val MyStruct) {
	d.Set("att", val.String)
}

func setStructFieldPointerString(d *ResourceData, val MyStruct) {
	d.Set("att", val.PointerString)
}

func setPointerStructFieldString(d *ResourceData, val *MyStruct) {
	d.Set("att", val.String)
}

func setPointerStructFieldPointerString(d *ResourceData, val *MyStruct) {
	d.Set("att", val.PointerString)
}

type MyString string

func setNamedString(d *ResourceData, val MyString) {
	d.Set("att", val)
}

type MyInt int

func setNamedInt(d *ResourceData, val MyInt) {
	d.Set("att", val)
}
`)
