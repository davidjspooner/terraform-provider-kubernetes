package tfparts

import (
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type LongMetadata struct {
	Namespace   basetypes.StringValue `tfsdk:"namespace"`
	Name        types.String          `tfsdk:"name"`
	Labels      types.Map             `tfsdk:"labels"`
	Annotations types.Map             `tfsdk:"annotations"`
}

func LongMetadataSchemaBlock() schema.SingleNestedBlock {
	s := schema.SingleNestedBlock{
		Attributes: map[string]schema.Attribute{
			"namespace": schema.StringAttribute{
				MarkdownDescription: "Namespace of the resource",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the resource",
				Required:            true,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Labels of the resource",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"annotations": schema.MapAttribute{
				MarkdownDescription: "Annotations of the resource",
				Optional:            true,
				ElementType:         types.StringType,
			},
		},
	}
	return s
}

func LongMetadataResourceAttr() map[string]rschema.Attribute {
	return map[string]rschema.Attribute{
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
				"labels": rschema.MapAttribute{
					MarkdownDescription: "Labels of the resource",
					Optional:            true,
					ElementType:         types.StringType,
				},
				"annotations": rschema.MapAttribute{
					MarkdownDescription: "Annotations of the resource",
					Optional:            true,
					ElementType:         types.StringType,
				},
			},
		},
	}
}

func LongMetadataDatasourceAttr() map[string]dschema.Attribute {
	return map[string]dschema.Attribute{
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
				"labels": dschema.MapAttribute{
					MarkdownDescription: "Labels of the resource",
					Optional:            true,
					ElementType:         types.StringType,
				},
				"annotations": dschema.MapAttribute{
					MarkdownDescription: "Annotations of the resource",
					Optional:            true,
					ElementType:         types.StringType,
				},
			},
		},
	}
}
