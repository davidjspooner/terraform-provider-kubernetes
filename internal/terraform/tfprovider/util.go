package tfprovider

import (
	"context"

	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type stringValueFunc struct {
	fn          func() string
	description string
}

func (r *stringValueFunc) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	s := r.fn()
	resp.PlanValue = types.StringValue(s)
	//resp.RequiresReplace = true
}
func (r *stringValueFunc) DefaultString(ctx context.Context, req defaults.StringRequest, resp *defaults.StringResponse) {
	s := r.fn()
	resp.PlanValue = types.StringValue(s)
}

func (r *stringValueFunc) Description(context.Context) string {
	return r.description
}
func (r *stringValueFunc) MarkdownDescription(context.Context) string {
	return r.description
}

func MergeResourceAttributes(attrs ...map[string]rschema.Attribute) map[string]rschema.Attribute {
	merged := make(map[string]rschema.Attribute)
	for _, attr := range attrs {
		for k, v := range attr {
			merged[k] = v
		}
	}
	return merged
}

func MergeDataAttributes(attrs ...map[string]dschema.Attribute) map[string]dschema.Attribute {
	merged := make(map[string]dschema.Attribute)
	for _, attr := range attrs {
		for k, v := range attr {
			merged[k] = v
		}
	}
	return merged
}
