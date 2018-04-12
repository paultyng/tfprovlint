package rules

import (
	"fmt"
	"os"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestDoNotDereferencePointersInSet(t *testing.T) {
	for i, c := range []struct {
		expectedMsg string
		funcName    string
	}{
		{"", "setConstString"},
		{"", "setVarString"},
		{"", "setCallString"},
		{"", "setPointer"},
		{"", "setReferenceString"},
		{"", "setReferencePointer"},
		{"", "alreadyInterface"},

		// TODO: test array and map access for false positives

		{"do not dereference value for attribute \"foo\" when calling d.Set", "setDereference"},
		{"do not dereference value for attribute \"foo\" when calling d.Set", "setReferenceDereferenec"},
	} {
		t.Run(fmt.Sprintf("%d %s", i, c.funcName), func(t *testing.T) {
			ci := lookupSetCallInstruction(c.funcName)
			if ci == nil {
				t.Fatalf("unable to find ssa.CallInstruction %q", c.funcName)
			}
			ci.Parent().WriteTo(os.Stdout)
			actualIssues, err := doNotDereferencePointersInSet(nil, nil, "foo", ci)
			if err != nil {
				t.Fatal(err)
			}
			if c.expectedMsg == "" {
				if len(actualIssues) > 0 {
					t.Fatalf("expected no issues but found %d", len(actualIssues))
				}
				return
			}
			if len(actualIssues) != 1 {
				t.Fatalf("expected only a single issue to be found (not %d)", len(actualIssues))
			}
			if msg := actualIssues[0].Message; msg != c.expectedMsg {
				t.Fatalf("unexpected message %q", msg)
			}
		})
	}
}

func lookupSetCallInstruction(funcName string) ssa.CallInstruction {
	f := resourceDataSetPkg.Func(funcName)
	if f == nil {
		panic(fmt.Sprintf("func %q not found", funcName))
	}
	for _, b := range f.Blocks {
		for _, ins := range b.Instrs {
			ssacall, ok := ins.(ssa.CallInstruction)
			if !ok {
				continue
			}
			if ssacall.Common().StaticCallee().Name() == "Set" {
				return ssacall
			}
		}
	}
	return nil
}

var resourceDataSetPkg = mustMakeSamplePkg(`
package test

type ResourceData struct {}

func (rd *ResourceData) Set(string, interface{}) error {
	// no-op
	return nil
}

func lookupString() string { return "" }

func setConstString(d *ResourceData) {
	d.Set("foo", "const")
}

func setVarString(d *ResourceData) {
	v := "a" + "b"
	d.Set("foo", v)
}

func setCallString(d *ResourceData) {
	d.Set("foo", lookupString())
}

func setDereference(d *ResourceData) {
	v1 := "foo"
	v2 := &v1
	d.Set("foo", *v2)
}

func setPointer(d *ResourceData) {
	v1 := "foo"
	v2 := &v1
	d.Set("foo", v2)
}

type TestStruct struct {
	String string
	PtrString *string
}

func setReferenceString(d *ResourceData) {
	instance := &TestStruct{}
	d.Set("foo", instance.String)
}

func setReferencePointer(d *ResourceData) {
	instance := &TestStruct{}
	d.Set("foo", instance.PtrString)
}

func setReferenceDereferenec(d *ResourceData) {
	instance := &TestStruct{}
	d.Set("foo", *instance.PtrString)
}

func alreadyInterface(d *ResourceData) {
	v1 := "foo"
	var v2 interface{} = v1
	d.Set("foo", v2)
}
`)
