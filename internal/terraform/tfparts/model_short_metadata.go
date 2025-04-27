package tfparts

import (
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type ShortMetadata struct {
	ApiVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
	Metadata   struct {
		Namespace basetypes.StringValue `tfsdk:"namespace"`
		Name      types.String          `tfsdk:"name"`
	} `tfsdk:"metadata"`
}

func ShortMetadataResourceAttr() map[string]rschema.Attribute {
	return map[string]rschema.Attribute{
		"api_version": rschema.StringAttribute{
			MarkdownDescription: "API version of the resource",
			Required:            true,
		},
		"kind": rschema.StringAttribute{
			MarkdownDescription: "Kind of the resource",
			Required:            true,
		},
		"metadata": rschema.SingleNestedAttribute{
			Description: "Metadata of the resource",
			Required:    true,
			Attributes: map[string]rschema.Attribute{
				"namespace": rschema.StringAttribute{
					MarkdownDescription: "Namespace of the resource",
					Optional:            true,
				},
				"name": rschema.StringAttribute{
					MarkdownDescription: "Name of the resource",
					Required:            true,
				},
			},
		},
	}
}

func ShortMetadataDatasourceAttr() map[string]dschema.Attribute {
	return map[string]dschema.Attribute{
		"api_version": dschema.StringAttribute{
			MarkdownDescription: "API version of the resource",
			Required:            true,
		},
		"kind": dschema.StringAttribute{
			MarkdownDescription: "Kind of the resource",
			Required:            true,
		},
		"metadata": dschema.SingleNestedAttribute{
			Description: "Metadata of the resource",
			Required:    true,
			Attributes: map[string]dschema.Attribute{
				"namespace": dschema.StringAttribute{
					MarkdownDescription: "Namespace of the resource",
					Optional:            true,
				},
				"name": dschema.StringAttribute{
					MarkdownDescription: "Name of the resource",
					Required:            true,
				},
			},
		},
	}
}
