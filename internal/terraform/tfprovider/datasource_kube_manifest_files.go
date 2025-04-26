package tfprovider

import (
	"context"

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
			tfTypeNameSuffix: "_manifest_files",
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
	FileNames types.List `tfsdk:"filenames"`
	Variables types.Map  `tfsdk:"variables"`
	Documents types.Map  `tfsdk:"documents"`
}

func (r *DataSourceKubeManifestFiles) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}

func (r *DataSourceKubeManifestFiles) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Read yaml from a list of files and return all the inner documents",

		Attributes: map[string]schema.Attribute{
			"filenames": schema.ListAttribute{
				MarkdownDescription: "List of paths to glob, load , split into documents, expand template and parse as a manifest",
				ElementType:         types.StringType,
				Required:            true,
			},
			"variables": schema.MapAttribute{
				MarkdownDescription: "Map of values to be used in the file. Requires template_type to be set",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"documents": schema.MapAttribute{
				MarkdownDescription: "results", //TODO
				ElementType: basetypes.ObjectType{
					AttrTypes: manifestMapElementAttrType,
				},
				Computed: true,
			},
		},
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
	// Retrieve the first argument as []string
	var filenames []string
	diags := config.FileNames.ElementsAs(ctx, &filenames, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Retrieve the second argument as map[string]any
	var variables map[string]string
	diags = config.Variables.ElementsAs(ctx, &variables, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	variables2 := make(map[string]any, len(variables))
	for k, v := range variables {
		variables2[k] = v
	}
	results, diags := SplitAndExpandTemplateFiles(filenames, variables2)
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
