package tfparts

import (
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/job"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kresource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	pschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type APIOptionsModel struct {
	Retry          *job.RetryModel `tfsdk:"retry"`
	FieldManager   *types.String   `tfsdk:"field_manager"`
	ForceConflicts *types.Bool     `tfsdk:"force_conflicts"`
}

func ApiOptionsResourceAttributes() map[string]rschema.Attribute {
	return map[string]rschema.Attribute{
		"api_options": rschema.SingleNestedAttribute{
			Description: "Options for the API request.",
			Required:    false,
			Attributes: map[string]rschema.Attribute{
				"retry": job.DefineRetryModelSchema(),
				"field_manager": rschema.StringAttribute{
					Description: "Field manager to use for the resource.",
					Optional:    true,
				},
				"force_conflicts": rschema.BoolAttribute{
					Description: "Force conflicts to be ignored.",
					Optional:    true,
				},
			},
			Optional: true,
		},
	}
}

func ApiOptionsDatasourceAttributes() map[string]dschema.Attribute {
	return map[string]dschema.Attribute{
		"api_options": dschema.SingleNestedAttribute{
			Description: "Options for the API request.",
			Required:    false,
			Attributes: map[string]dschema.Attribute{
				"retry": job.DefineRetryModelSchema(),
				"field_manager": dschema.StringAttribute{
					Description: "Field manager to use for the resource.",
					Optional:    true,
				},
				"force_conflicts": dschema.BoolAttribute{
					Description: "Force conflicts to be ignored.",
					Optional:    true,
				},
			},
			Optional: true,
		},
	}
}

func ApiOptionProviderAttributes() map[string]pschema.Attribute {
	return map[string]pschema.Attribute{
		"api_options": pschema.SingleNestedAttribute{
			Description: "Options for the API request.",
			Required:    false,
			Attributes: map[string]pschema.Attribute{
				"retry": job.DefineRetryModelSchema(),
				"field_manager": pschema.StringAttribute{
					Description: "Field manager to use for the resource.",
					Optional:    true,
				},
				"force_conflicts": pschema.BoolAttribute{
					Description: "Force conflicts to be ignored.",
					Optional:    true,
				},
			},
			Optional: true,
		},
	}
}

func (model *APIOptionsModel) Options() *kresource.APIClientOptions {
	if model == nil {
		return nil
	}
	opt := &kresource.APIClientOptions{
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
