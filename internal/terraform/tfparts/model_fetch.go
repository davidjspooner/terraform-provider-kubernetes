package tfparts

import (
	"fmt"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kube"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type FetchItem struct {
	Field types.String `tfsdk:"field"`
	Match types.String `tfsdk:"match"`
}

type FetchMap struct {
	Fetch  types.Map `tfsdk:"fetch"`
	Output types.Map `tfsdk:"output"`
}

func FetchRequestAttributes() map[string]rschema.Attribute {
	return map[string]rschema.Attribute{
		"fetch": rschema.MapNestedAttribute{
			Description: "Map of fields to check. ( key name will be used in output )",
			Optional:    true,
			NestedObject: rschema.NestedAttributeObject{
				Attributes: map[string]rschema.Attribute{
					"field": rschema.StringAttribute{
						Description: "Path to the field to check.",
						Required:    true,
					},
					"match": rschema.StringAttribute{
						Description: "Regular expression to match the field.",
						Optional:    true,
					},
				},
			},
		},
		"output": rschema.MapAttribute{
			Description: "Map of queries to check.",
			ElementType: types.StringType,
			Computed:    true,
		},
	}
}

func FetchDatasourceAttributes(required bool) map[string]dschema.Attribute {
	return map[string]dschema.Attribute{
		"fetch": dschema.ListNestedAttribute{
			Description: "List of queries to check.",
			Required:    required,
			NestedObject: dschema.NestedAttributeObject{
				Attributes: map[string]dschema.Attribute{
					"field": dschema.StringAttribute{
						Description: "Path to the field to check.",
						Required:    true,
					},
					"match": dschema.StringAttribute{
						Description: "Regular expression to match the field.",
						Optional:    true,
					},
				},
			},
			Optional: true,
		},
		"output": rschema.MapAttribute{
			Description: "Map of queries to check.",
			ElementType: types.StringType,
			Computed:    true,
		},
	}
}

func (w *FetchMap) Compile() (*kube.CompiledFetchMap, error) {
	if w == nil {
		return nil, nil
	}
	compiled := kube.CompiledFetchMap{}
	for key, value := range w.Fetch.Elements() {
		object, ok := value.(basetypes.ObjectValue)
		if !ok {
			return nil, fmt.Errorf("expected ObjectValue, got %T", value)
		}
		_ = object

		attr := object.Attributes()
		fieldAttr, ok := attr["field"]
		if !ok {
			return nil, fmt.Errorf("missing field attribute")
		}
		field, ok := fieldAttr.(types.String)
		if !ok {
			return nil, fmt.Errorf("expected String, got %T", fieldAttr)
		}
		fieldString := field.ValueString()
		var pattern string
		matchAttr, ok := attr["match"]
		if ok {
			match, ok := matchAttr.(types.String)
			if !ok {
				return nil, fmt.Errorf("expected String, got %T", matchAttr)
			}
			pattern = match.ValueString()
		}
		err := compiled.Add(key, fieldString, pattern)
		if err != nil {
			return nil, fmt.Errorf("error adding fetch item %s: %w", key, err)
		}
	}
	return &compiled, nil
}
