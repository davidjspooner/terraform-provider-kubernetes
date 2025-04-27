package tfprovider

import (
	"context"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/terraform/tfparts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DataSourceKubeManifestFiles{}

func init() {
	// Register the data source with the provider.
	RegisterDataSource(func() datasource.DataSource {
		ds := &DataSourceKubeManifestFiles{
			tfTypeNameSuffix: "_manifest_documents",
		}
		return ds
	})
}

// DataSourceKubeManifestFiles defines the resource implementation.
type DataSourceKubeManifestFiles struct {
	tfTypeNameSuffix string
}

// FileManifestsModel describes the resource data tfshared.
type FileManifestsModel struct {
	tfparts.FileSetModelList
	Documents types.Map `tfsdk:"documents"`
}

func (r *DataSourceKubeManifestFiles) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}

func (r *DataSourceKubeManifestFiles) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attr := map[string]schema.Attribute{
		"documents": schema.MapAttribute{
			MarkdownDescription: "results", //TODO
			ElementType: basetypes.ObjectType{
				AttrTypes: tfparts.DocumentElementAttrType,
			},
			Computed: true,
		},
	}
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Read yaml from a list of files and return all the inner documents",

		Attributes: MergeDataAttributes(
			attr,
			tfparts.FileSetsDatasourceAttributes(true),
		),
	}
}

func (r *DataSourceKubeManifestFiles) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
}

func (r *DataSourceKubeManifestFiles) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config FileManifestsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fsds := config.GetFileSetDefs()
	for i := range fsds {
		fsds[i].SplitYamlDocs = true
	}

	results, diags := tfparts.GenerateDocumentList(fsds)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	config.Documents = results

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
	if diags.HasError() {
		return
	}
}
