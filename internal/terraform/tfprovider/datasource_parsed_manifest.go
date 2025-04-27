package tfprovider

import (
	"context"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kube"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/terraform/tfparts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DataSourceManifest{}

func init() {
	// Register the data source with the provider.
	RegisterDataSource(func() datasource.DataSource {
		ds := &DataSourceManifest{
			tfTypeNameSuffix: "_parsed_manifest",
		}
		return ds
	})
}

// DataSourceManifest defines the resource implementation.
type DataSourceManifest struct {
	tfTypeNameSuffix string
}

// FileManifestsModel describes the resource data tfshared.
type ManifestsModel struct {
	Text     types.String  `tfsdk:"text"`
	Manifest types.Dynamic `tfsdk:"manifest"`
}

func (r *DataSourceManifest) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}

func (r *DataSourceManifest) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Read yaml from a list of files and return all the inner manifests",

		Attributes: map[string]schema.Attribute{
			"text": schema.StringAttribute{
				MarkdownDescription: "yaml formatted manifest",
				Required:            true,
			},
			"manifest": schema.DynamicAttribute{
				MarkdownDescription: "the manifest as an object",
				Computed:            true,
			},
		},
	}
}

func (r *DataSourceManifest) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
}

func (r *DataSourceManifest) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ManifestsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	text := config.Text.ValueString()
	if text == "" {
		resp.Diagnostics.AddError("No text provided", "The text attribute must be set to a non-empty string.")
		return
	}

	u, err := kube.ParseSingleYamlManifest(text)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing yaml", err.Error())
		return
	}
	// Convert the unstructured object to a map[string]any
	manifest, err := tfparts.UnstructuredToDynamic(u)
	if err != nil {
		resp.Diagnostics.AddError("Error converting unstructured to dynamic", err.Error())
		return
	}
	config.Manifest = manifest
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
