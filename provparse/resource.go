package provparse

import (
	"go/token"

	"golang.org/x/tools/go/ssa"
)

// Provider represents the data for the provider.
type Provider struct {
	Name        string
	Resources   []Resource
	DataSources []Resource
	Fset        *token.FileSet
}

func findResource(resources []Resource, name string) *Resource {
	for _, r := range resources {
		if r.Name == name {
			return &r
		}
	}

	return nil
}

// Resource looks up a resource within the provider by name and returns nil if not found.
func (p *Provider) Resource(name string) *Resource {
	return findResource(p.Resources, name)
}

// DataSource looks up a data source in the provider by name and returns nil if not found.
func (p *Provider) DataSource(name string) *Resource {
	return findResource(p.DataSources, name)
}

// Resource represents the data for a resource or data source of the provider.
type Resource struct {
	Name        string
	FuncComment string

	Func       *ssa.Function
	CreateFunc *ssa.Function
	ReadFunc   *ssa.Function
	UpdateFunc *ssa.Function
	DeleteFunc *ssa.Function
	ExistsFunc *ssa.Function

	Attributes []Attribute

	// PartialParse indicates that there is a high probability the full details were not read
	// for the resource.
	PartialParse bool
	// TODO: start setting this flag where warnings are logged
}

func findAttribute(atts []Attribute, name string) *Attribute {
	for _, att := range atts {
		if att.Name == name {
			return &att
		}
	}

	return nil
}

// Attribute returns an attribute of the resource by name or nil if not found.
func (r *Resource) Attribute(name string) *Attribute {
	return findAttribute(r.Attributes, name)
}

// Attribute returns a child attribute by name or nil if not found.
func (a *Attribute) Attribute(name string) *Attribute {
	return findAttribute(a.Attributes, name)
}

// Attribute represents a data element of the resource schema.
type Attribute struct {
	Name        string
	Description string

	Optional bool
	Required bool
	Computed bool

	Type AttributeType

	Attributes []Attribute
}

// AttributeType maps roughly to helper/schema.ValueType
type AttributeType int

// These constants map roughly to the values for helper/schema.ValueType
const (
	TypeInvalid AttributeType = iota
	TypeBool
	TypeInt
	TypeFloat
	TypeString
	TypeList
	TypeMap
	TypeSet
)
