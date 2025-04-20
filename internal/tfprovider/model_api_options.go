package tfprovider

import (
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/job"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type APIOptionsModel struct {
	Retry          *job.RetryModel `tfsdk:"retry"`
	FieldManager   *types.String   `tfsdk:"field_manager"`
	ForceConflicts *types.Bool     `tfsdk:"force_conflicts"`
}

func ApiOptionsModelSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: "Options for the API request.",
		Required:    false,
		Attributes: map[string]schema.Attribute{
			"retry": job.DefineRetryModelSchema(),
			"field_manager": schema.StringAttribute{
				Description: "Field manager to use for the resource.",
				Optional:    true,
			},
			"force_conflicts": schema.BoolAttribute{
				Description: "Force conflicts to be ignored.",
				Optional:    true,
			},
		},
	}
}

func (model *APIOptionsModel) Options() *kresource.APIOptions {
	opt := &kresource.APIOptions{
		Retry: model.Retry,
	}
	if model.FieldManager != nil {
		s := model.FieldManager.ValueString()
		if s != "" {
			opt.FieldManager = &s
		}
	}
	if model.ForceConflicts != nil {
		b := model.ForceConflicts.ValueBool()
		opt.ForceConflicts = &b
	}
	return opt
}
