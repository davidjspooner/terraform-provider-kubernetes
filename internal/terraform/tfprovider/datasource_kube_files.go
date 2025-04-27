package tfprovider

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kube"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/terraform/tfparts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DataSourceKubeFiles{}

func init() {
	// Register the data source with the provider.
	RegisterDataSource(func() datasource.DataSource {
		ds := &DataSourceKubeFiles{
			tfTypeNameSuffix: "_files",
		}
		return ds
	})
}

// DataSourceKubeFiles defines the resource implementation.
type DataSourceKubeFiles struct {
	tfTypeNameSuffix string
}

// FileManifestsModel describes the resource data tfshared.
type FileModel struct {
	tfparts.FileSetModelList
	Contents types.Map `tfsdk:"contents"`
}

func (r *DataSourceKubeFiles) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}

func (r *DataSourceKubeFiles) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attr := map[string]schema.Attribute{
		"contents": schema.MapAttribute{
			MarkdownDescription: "results", //TODO
			ElementType:         types.StringType,
			Computed:            true,
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

func (r *DataSourceKubeFiles) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
}

func (r *DataSourceKubeFiles) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config FileModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fsds := config.GetFileSetDefs()
	contents := make(map[string]attr.Value)

	var handler kube.ExpandedContentHandlerFunc = func(content *kube.ExpandedContent) error {
		basename := filepath.Base(content.Filename)
		_, exists := contents[basename]
		if exists {
			return fmt.Errorf("duplicate name %s", basename)
		}
		contents[basename] = types.StringValue(string(content.Content))
		return nil
	}
	err := fsds.ExpandContent(handler)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error expanding file set",
			fmt.Sprintf("Error expanding file set: %s", err),
		)
		return
	}
	var diags diag.Diagnostics
	config.Contents, diags = basetypes.NewMapValue(types.StringType, contents)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
	if diags.HasError() {
		return
	}
}
