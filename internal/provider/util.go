package provider

import (
	"context"

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
