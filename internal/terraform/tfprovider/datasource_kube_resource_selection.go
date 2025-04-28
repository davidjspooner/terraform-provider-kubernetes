package tfprovider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type DataSourceResourceSelector struct {
	tfTypeNameSuffix string
	schema           schema.Schema
	provider         *KubeProvider
}

var _ datasource.DataSource = &DataSourceResourceSelector{}

type ResourceMetadata struct {
	ApiVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
	Metadata   struct {
		Namespace   basetypes.StringValue `tfsdk:"namespace"`
		Name        types.String          `tfsdk:"name"`
		Labels      map[string]string     `tfsdk:"labels"`
		Annotations map[string]string     `tfsdk:"annotations"`
	} `tfsdk:"metadata"`
}

func (r *ResourceMetadata) String() string {
	return fmt.Sprintf("%s:%s:%s (%s)", r.Kind.ValueString(), r.Metadata.Namespace.ValueString(), r.Metadata.Name.ValueString(), r.ApiVersion.ValueString())
}

type ResourceSelectorModel struct {
	Kinds      []string          `tfsdk:"kinds"`
	Namespaces []string          `tfsdk:"namespaces"`
	Labels     map[string]string `tfsdk:"labels"`
	Resources  types.Map         `tfsdk:"resources"`
}

func init() {
	// Register the data source with the provider.
	RegisterDataSource(func() datasource.DataSource {
		attr := map[string]schema.Attribute{
			"kinds": schema.ListAttribute{
				MarkdownDescription: "The kinds of resources to select",
				ElementType:         types.StringType,
				Required:            true,
			},
			"namespaces": schema.ListAttribute{
				MarkdownDescription: "The namespaces to select",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "The labels to select",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"resources": schema.MapAttribute{
				MarkdownDescription: "The resources that match the selector",
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"api_version": types.StringType,
						"kind":        types.StringType,
						"metadata": types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"namespace":   types.StringType,
								"name":        types.StringType,
								"labels":      types.MapType{ElemType: types.StringType},
								"annotations": types.MapType{ElemType: types.StringType},
							},
						},
					},
				},
				Computed: true,
			},
		}
		ds := &DataSourceResourceSelector{
			tfTypeNameSuffix: "_resource_selection",
			schema: schema.Schema{
				Description: "Get the resource selector from the kubeconfig file",
				Attributes:  attr,
			},
		}
		return ds
	})
}
func (r *DataSourceResourceSelector) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}
func (r *DataSourceResourceSelector) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = r.schema
}

func (r *DataSourceResourceSelector) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	provider, ok := req.ProviderData.(*KubeProvider)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Type", "Expected provider data to be of type *KubernetesResourceProvider")
		return
	}
	r.provider = provider

}
func (r *DataSourceResourceSelector) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Get the provider config
	var state ResourceSelectorModel
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	var namespaces []string
	if len(state.Namespaces) > 0 {
		namespaces = append(namespaces, state.Namespaces...)
	} else {
		namespaces = []string{"*"}
	}
	_ = namespaces

	var kinds []string
	if len(state.Kinds) > 0 {
		kinds = append(kinds, state.Kinds...)
	} else {
		kinds = []string{"*"}
	}
	_ = kinds

	resp.Diagnostics.Append(diag.NewErrorDiagnostic("not implemented", "The resource selector is not implemented yet"))
}
