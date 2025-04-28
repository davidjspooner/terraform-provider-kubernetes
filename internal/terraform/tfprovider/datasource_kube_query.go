package tfprovider

import (
	"context"
	"fmt"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kube"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/terraform/tfparts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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
				map[string]schema.Attribute{},
				tfparts.FetchDatasourceAttributes(false),
				tfparts.ApiOptionsDatasourceAttributes(),
				tfparts.ShortMetadataDatasourceAttr(),
			),
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
	tfparts.ShortMetadata
	tfparts.APIOptionsModel
	tfparts.FetchMap
}

func (m *KubeQueryModel) GetResouceKey() (kube.ResourceKey, error) {
	rk := kube.ResourceKey{
		ApiVersion: m.ApiVersion.ValueString(),
		Kind:       m.Kind.ValueString(),
	}
	rk.Metadata.Name = m.Metadata.Name.ValueString()
	namespace := m.Metadata.Namespace.ValueString()
	if namespace != "" {
		rk.Metadata.Namespace = &namespace
	}
	return rk, nil
}
func (m *KubeQueryModel) BuildManifest(manifest *unstructured.Unstructured) error {
	return fmt.Errorf("BuildManifest not implemented")
}

func (m *KubeQueryModel) UpdateFrom(manifest unstructured.Unstructured) error {
	return nil
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
