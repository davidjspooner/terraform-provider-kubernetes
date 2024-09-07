package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ShortMetadataModel struct {
	Namespace types.String `tfsdk:"namespace"`
	Name      types.String `tfsdk:"name"`
}

func ShortMetadataSchemaBlock() schema.SingleNestedBlock {
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
		},
	}
	return s
}

type LongMetadataModel struct {
	Namespace   types.String `tfsdk:"namespace"`
	Name        types.String `tfsdk:"name"`
	Labels      types.Map    `tfsdk:"labels"`
	Annotations types.Map    `tfsdk:"annotations"`
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
