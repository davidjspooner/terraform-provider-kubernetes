package tfprovider

import (
	"context"
	"fmt"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kube"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/terraform/tfparts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DataSourceKubeQuery{}

func init() {
	// Register the data source with the provider.
	RegisterDataSource(func() datasource.DataSource {
		ds := &DataSourceKubeQuery{}
		ds.tfTypeNameSuffix = "_query"
		ds.schema = schema.Schema{
			MarkdownDescription: "Query a Kubernetes resource and fetch specific fields.",
			Attributes: MergeDataAttributes(
				map[string]schema.Attribute{
					"after": schema.DynamicAttribute{
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

		return ds
	})
}

// DataSourceKubeQuery defines the datasource implementation.
type DataSourceKubeQuery struct {
	DataSourceBase[*KubeQueryModel]
}

// KubeQueryModel describes the datasource data model.
type KubeQueryModel struct {
	DependsOn  types.Dynamic          `tfsdk:"after"`
	ApiVersion types.String           `tfsdk:"api_version"`
	Kind       types.String           `tfsdk:"kind"`
	Metadata   *tfparts.ShortMetadata `tfsdk:"metadata"`
	tfparts.APIOptionsModel
	tfparts.FetchMap
}

func (m *KubeQueryModel) GetResouceKey() (kube.ResourceKey, error) {
	return kube.ResourceKey{}, fmt.Errorf("GetResouceKey not implemented")
}
func (m *KubeQueryModel) BuildManifest(manifest *unstructured.Unstructured) error {
	return fmt.Errorf("BuildManifest not implemented")
}

func (m *KubeQueryModel) UpdateFrom(manifest unstructured.Unstructured) error {
	//TODO: Implement this method to update the model from the manifest
	return fmt.Errorf("UpdateFrom not implemented")
}

func (d *DataSourceKubeQuery) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	d.DataSourceBase.Metadata(ctx, req, resp)
}

func (d *DataSourceKubeQuery) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	d.DataSourceBase.Schema(ctx, req, resp)
}

func (d *DataSourceKubeQuery) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.DataSourceBase.Configure(ctx, req, resp)
}

func (d *DataSourceKubeQuery) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data KubeQueryModel
	d.DataSourceBase.Read(ctx, &data, req, resp)
}
