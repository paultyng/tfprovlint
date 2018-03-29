package provparse

type Provider struct {
	Resources   []Resource
	DataSources []Resource
}

func findResource(resources []Resource, name string) *Resource {
	for _, r := range resources {
		if r.Name == name {
			return &r
		}
	}

	return nil
}

func (p *Provider) Resource(name string) *Resource {
	return findResource(p.Resources, name)
}

func (p *Provider) DataSource(name string) *Resource {
	return findResource(p.DataSources, name)
}

type Resource struct {
	Provider         string // azurerm
	Name             string // azurerm_image
	NameSuffix       string // image
	Type             string // data vs resource?
	ShortDescription string // Get information about an Image
	Description      string // Use this data source to access information about an Image.
	// TODO: +Example usage, etc?
	// TODO: resource category
	Attributes []Attribute
}

func findAttribute(atts []Attribute, name string) *Attribute {
	for _, att := range atts {
		if att.Name == name {
			return &att
		}
	}

	return nil
}

func (r *Resource) Attribute(name string) *Attribute {
	return findAttribute(r.Attributes, name)
}

func (a *Attribute) Attribute(name string) *Attribute {
	return findAttribute(a.Attributes, name)
}

type Attribute struct {
	Name        string
	Description string
	Optional    bool
	Required    bool
	Computed    bool

	Attributes []Attribute
	Min        int
	Max        int
}
