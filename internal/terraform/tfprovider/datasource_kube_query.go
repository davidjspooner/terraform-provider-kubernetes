package tfprovider

import (
	"context"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kube"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/terraform/tfparts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DataSourceKubeQuery{}

func init() {
	// Register the data source with the provider.
	RegisterDataSource(func() datasource.DataSource {
		return &DataSourceKubeQuery{
			tfTypeNameSuffix: "_query",
		}
	})
}

// DataSourceKubeQuery defines the datasource implementation.
type DataSourceKubeQuery struct {
	provider         *KubernetesResourceProvider
	tfTypeNameSuffix string
}

// KubeQueryModel describes the datasource data model.
type KubeQueryModel struct {
	DependsOn  types.Set              `tfsdk:"depends_on"`
	ApiVersion types.String           `tfsdk:"api_version"`
	Kind       types.String           `tfsdk:"kind"`
	Metadata   *tfparts.ShortMetadata `tfsdk:"metadata"`
	tfparts.APIOptionsModel
	tfparts.FetchMap
}

func (d *DataSourceKubeQuery) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + d.tfTypeNameSuffix
}

func (d *DataSourceKubeQuery) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Query a Kubernetes resource and fetch specific fields.",
		Attributes: MergeDataAttributes(
			map[string]schema.Attribute{
				"depends_on": schema.SetAttribute{
					MarkdownDescription: "Resources that this resource depends on.",
					Optional:            true,
				},
				"api_version": schema.StringAttribute{
					MarkdownDescription: "API version of the resource.",
					Required:            true,
				},
				"kind": schema.StringAttribute{
					MarkdownDescription: "Kind of the resource.",
					Required:            true,
				},
			},
			tfparts.FetchDatasourceAttributes(false),
			tfparts.ApiOptionsDatasourceAttributes(),
		),
		Blocks: map[string]schema.Block{
			"metadata": tfparts.ShortMetadataSchemaBlock(),
		},
	}
}

func (d *DataSourceKubeQuery) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var ok bool
	d.provider, ok = req.ProviderData.(*KubernetesResourceProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			"Expected *KubernetesResourceProvider. Please report this issue to the provider developers.",
		)
	}
}

func (d *DataSourceKubeQuery) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data KubeQueryModel

	// Read Terraform configuration into the model.
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create a resource helper.
	resourceOptions := GetPtrToEmbedddedType[tfparts.APIOptionsModel](&data)
	options, err := kube.MergeAPIOptions(d.provider.DefaultApiOptions, resourceOptions.Options())
	if err != nil {
		resp.Diagnostics.AddError("Failed to merge API options", err.Error())
		return
	}

	key := kube.ResourceKey{
		ApiVersion: data.ApiVersion.ValueString(),
		Kind:       data.Kind.ValueString(),
	}
	key.MetaData.Name = data.Metadata.Name.ValueString()
	s := data.Metadata.Namespace.ValueString()
	if s != "" {
		key.MetaData.Namespace = &s
	}
	resourceHelper, err := kube.NewResourceHelper(ctx, &d.provider.Shared, options, key)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource helper", err.Error())
		return
	}

	// Compile the fetch map.
	compiledFetch, err := data.FetchMap.Compile()
	if err != nil {
		resp.Diagnostics.AddError("Failed to compile fetch map", err.Error())
		return
	}

	// Fetch the data.
	outputs, err := resourceHelper.Fetch(ctx, nil, compiledFetch, kube.MayOrMayNotExist)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch resource data", err.Error())
		return
	}

	// Set the output map.
	outputMap := make(map[string]types.String)
	for k, v := range outputs {
		outputMap[k] = types.StringValue(v)
	}
	data.Output, _ = types.MapValueFrom(ctx, types.StringType, outputMap)

	// Save the state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
