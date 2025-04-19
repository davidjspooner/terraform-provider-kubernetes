package tfprovider

import (
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ShortMetadata struct {
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

type OutputMetadata struct {
	ResourceVersion types.String `tfsdk:"resource_version"`
	UID             types.String `tfsdk:"uid"`
	Generation      types.Int64  `tfsdk:"generation"`
}

func (output *OutputMetadata) FromManifest(manifest *unstructured.Unstructured) {
	s, found, err := unstructured.NestedString(manifest.Object, "metadata", "resourceVersion")
	if err != nil || !found {
		s = ""
	}
	output.ResourceVersion = basetypes.NewStringValue(s)

	s, found, err = unstructured.NestedString(manifest.Object, "metadata", "uid")
	if err != nil || !found {
		s = ""
	}
	output.UID = basetypes.NewStringValue(s)

	n, found, err := unstructured.NestedInt64(manifest.Object, "metadata", "generation")
	if err != nil || !found {
		n = 0
	}
	output.Generation = basetypes.NewInt64Value(n)
}
