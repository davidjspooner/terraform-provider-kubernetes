package tfprovider

import (
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/job"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type APIOptions struct {
	Retry          *job.RetryModel `tfsdk:"retry"`
	FieldManager   *types.String   `tfsdk:"field_manager"`
	ForceConflicts *types.Bool     `tfsdk:"force_conflicts"`
}

func MergeKubenetesAPIOptions(
	models ...*APIOptions,
) (*APIOptions, error) {
	merged := &APIOptions{}
	var err error
	for _, model := range models {
		if model.Retry != nil {
			merged.Retry, err = job.MergeRetryModels(merged.Retry, model.Retry)
			if err != nil {
				return nil, err
			}
		}
		if model.FieldManager != nil {
			merged.FieldManager = model.FieldManager
		}
		if model.ForceConflicts != nil {
			merged.ForceConflicts = model.ForceConflicts
		}
	}
	return merged, nil
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
