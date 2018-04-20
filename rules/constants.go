package rules

// well known SDK funcs
const (
	funcResourceDataSetId = "(*github.com/hashicorp/terraform/helper/schema.ResourceData).SetId"
	funcResourceDataSet   = "(*github.com/hashicorp/terraform/helper/schema.ResourceData).Set"

	funcRemoveFromState = "github.com/hashicorp/terraform/helper/schema.RemoveFromState"
)
