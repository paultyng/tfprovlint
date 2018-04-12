package rules

import (
	"fmt"
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
		{"", "setValueString"},
		{"", "setValuePointer"},
		{"", "alreadyInterface"},
		{"", "setFromSlice"},
		{"", "setFromPointerSlice"},
		{"", "setFromMap"},
		{"", "setFromPointerMap"},
		{"", "setCallPointerString"},

		{"do not dereference value for attribute \"foo\" when calling d.Set", "setDereference"},
		{"do not dereference value for attribute \"foo\" when calling d.Set", "setCallPointerStringDereference"},
		{"do not dereference value for attribute \"foo\" when calling d.Set", "setReferenceDereference"},
		{"do not dereference value for attribute \"foo\" when calling d.Set", "setValueDereference"},
		{"do not dereference value for attribute \"foo\" when calling d.Set", "setFromSliceDereference"},
		{"do not dereference value for attribute \"foo\" when calling d.Set", "setFromMapDereference"},
	} {
		t.Run(fmt.Sprintf("%d %s", i, c.funcName), func(t *testing.T) {
			ci := lookupSetCallInstruction(c.funcName)
			if ci == nil {
				t.Fatalf("unable to find ssa.CallInstruction %q", c.funcName)
			}
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

func setConstString(d *ResourceData) {
	d.Set("foo", "const")
}

func setVarString(d *ResourceData) {
	v := "a" + "b"
	d.Set("foo", v)
}

func lookupString() string { return "" }

func setCallString(d *ResourceData) {
	d.Set("foo", lookupString())
}

func lookupStringPointer() *string {
	s := "bar"
	return &s
}

func setCallPointerString(d *ResourceData) {
	d.Set("foo", lookupStringPointer())
}

func setCallPointerStringDereference(d *ResourceData) {
	d.Set("foo", *lookupStringPointer())
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

func setReferenceDereference(d *ResourceData) {
	instance := &TestStruct{}
	d.Set("foo", *instance.PtrString)
}

func setValueString(d *ResourceData) {
	instance := TestStruct{}
	d.Set("foo", instance.String)
}

func setValuePointer(d *ResourceData) {
	instance := TestStruct{}
	d.Set("foo", instance.PtrString)
}

func setValueDereference(d *ResourceData) {
	instance := TestStruct{}
	d.Set("foo", *instance.PtrString)
}

func alreadyInterface(d *ResourceData) {
	v1 := "foo"
	var v2 interface{} = v1
	d.Set("foo", v2)
}

func setFromSlice(d *ResourceData) {
	sl := []string{"a"}
	d.Set("foo", sl[0])
}

func setFromPointerSlice(d *ResourceData) {
	sl := []*string{}
	d.Set("foo", sl[0])
}

func setFromSliceDereference(d *ResourceData) {
	sl := []*string{}
	d.Set("foo", *sl[0])
}

func setFromMap(d *ResourceData) {
	m := map[int]string{1: "a"}
	d.Set("foo", m[1])
}

func setFromPointerMap(d *ResourceData) {
	m := map[int]*string{}
	d.Set("foo", m[1])
}

func setFromMapDereference(d *ResourceData) {
	m := map[int]*string{}
	d.Set("foo", *m[1])
}
`)
